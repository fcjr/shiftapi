package shiftapi

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

var pathParamRe = regexp.MustCompile(`\{([^}]+)\}`)

func (a *API) updateSchema(method, path string, pathType, queryType, headerType, inType, outType reflect.Type, hasRespHeader, noBody, hasForm bool, formType reflect.Type, info *RouteInfo, status int, errors []errorEntry, staticHeaders []staticResponseHeader) error {
	op := &openapi3.Operation{
		OperationID: operationID(method, path),
		Responses:   openapi3.NewResponses(),
	}

	// Build a map from path param name to struct field for typed path params.
	pathFields := make(map[string]reflect.StructField)
	if pathType != nil {
		pt := pathType
		for pt.Kind() == reflect.Pointer {
			pt = pt.Elem()
		}
		if pt.Kind() == reflect.Struct {
			for f := range pt.Fields() {
				if f.IsExported() && hasPathTag(f) {
					pathFields[pathFieldName(f)] = f
				}
			}
		}
	}

	// Path parameters
	for _, match := range pathParamRe.FindAllStringSubmatch(path, -1) {
		name := match[1]
		var schema *openapi3.SchemaRef
		if field, ok := pathFields[name]; ok {
			schema = scalarToOpenAPISchema(field.Type)
			_ = validateSchemaCustomizer(name, field.Type, field.Tag, schema.Value)
		} else {
			schema = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: &openapi3.Types{"string"},
				},
			}
		}
		op.Parameters = append(op.Parameters, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:     name,
				In:       "path",
				Required: true,
				Schema:   schema,
			},
		})
	}

	// Query parameters
	if queryType != nil {
		queryParams, err := a.generateQueryParams(queryType)
		if err != nil {
			return err
		}
		op.Parameters = append(op.Parameters, queryParams...)
	}

	// Header parameters
	if headerType != nil {
		headerParams, err := a.generateHeaderParams(headerType)
		if err != nil {
			return err
		}
		op.Parameters = append(op.Parameters, headerParams...)
	}

	// Response schema
	statusStr := fmt.Sprintf("%d", status)

	// Build response header definitions from header-tagged fields on the output type
	// and static response headers from WithResponseHeader.
	var respHeaders openapi3.Headers
	if hasRespHeader && outType != nil {
		respHeaders = generateRespHeaders(outType)
	}
	for _, h := range staticHeaders {
		if respHeaders == nil {
			respHeaders = make(openapi3.Headers)
		}
		respHeaders[h.name] = &openapi3.HeaderRef{
			Value: &openapi3.Header{
				Parameter: openapi3.Parameter{
					Name:     h.name,
					In:       "header",
					Required: true,
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"string"},
						},
					},
				},
			},
		}
	}

	if noBody {
		// No-body status codes (204, 304) — emit response with description
		// and optional headers, but no content.
		resp := &openapi3.Response{
			Description: new(http.StatusText(status)),
		}
		if len(respHeaders) > 0 {
			resp.Headers = respHeaders
		}
		op.Responses.Set(statusStr, &openapi3.ResponseRef{Value: resp})
	} else {
		outSchema, err := a.generateSchemaRef(outType)
		if err != nil {
			return err
		}
		if hasRespHeader && outSchema != nil {
			stripRespHeaderFields(outType, outSchema.Value)
		}

		if outSchema != nil {
			resp := &openapi3.Response{
				Description: new(http.StatusText(status)),
			}
			if len(outSchema.Value.Properties) > 0 {
				resp.Content = map[string]*openapi3.MediaType{
					"application/json": {
						Schema: &openapi3.SchemaRef{
							Ref: fmt.Sprintf("#/components/schemas/%s", outSchema.Ref),
						},
					},
				}
				a.spec.Components.Schemas[outSchema.Ref] = &openapi3.SchemaRef{
					Value: outSchema.Value,
				}
			}
			if len(respHeaders) > 0 {
				resp.Headers = respHeaders
			}
			op.Responses.Set(statusStr, &openapi3.ResponseRef{Value: resp})
		} else if len(respHeaders) > 0 {
			op.Responses.Set(statusStr, &openapi3.ResponseRef{
				Value: &openapi3.Response{
					Description: new(http.StatusText(status)),
					Headers:     respHeaders,
				},
			})
		}
	}

	// Error responses — always include 400, 422, and 500.
	op.Responses.Set("400", errorResponseRef("Bad Request", "BadRequestError"))
	op.Responses.Set("422", errorResponseRef("Validation Error", "ValidationError"))
	op.Responses.Set("500", errorResponseRef("Internal Server Error", "InternalServerError"))

	// Add user-declared error responses from WithError.
	for _, e := range errors {
		codeStr := fmt.Sprintf("%d", e.status)
		errSchema, err := a.generateSchemaRef(e.typ)
		if err != nil {
			return err
		}
		if errSchema != nil && errSchema.Ref != "" {
			a.spec.Components.Schemas[errSchema.Ref] = &openapi3.SchemaRef{
				Value: errSchema.Value,
			}
			op.Responses.Set(codeStr, errorResponseRef(
				http.StatusText(e.status),
				errSchema.Ref,
			))
		}
	}

	// Request body schema
	if hasForm {
		// multipart/form-data request body
		formSchema, formEncoding := generateFormSchema(formType)
		mediaType := &openapi3.MediaType{
			Schema: &openapi3.SchemaRef{
				Value: formSchema,
			},
		}
		if formEncoding != nil {
			mediaType.Encoding = formEncoding
		}
		op.RequestBody = &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Required: true,
				Content: map[string]*openapi3.MediaType{
					"multipart/form-data": mediaType,
				},
			},
		}
	} else if inType != nil {
		inSchema, err := a.generateSchemaRef(inType)
		if err != nil {
			return err
		}
		if inSchema != nil {
			// Strip query-tagged, header-tagged, and path-tagged fields from the body schema
			stripQueryFields(inType, inSchema.Value)
			stripHeaderFields(inType, inSchema.Value)
			stripPathFields(inType, inSchema.Value)

			if len(inSchema.Value.Properties) > 0 {
				// Named body schema with properties
				content := make(map[string]*openapi3.MediaType)
				content["application/json"] = &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Ref: fmt.Sprintf("#/components/schemas/%s", inSchema.Ref),
					},
				}
				op.RequestBody = &openapi3.RequestBodyRef{
					Value: &openapi3.RequestBody{
						Required: true,
						Content:  content,
					},
				}
				a.spec.Components.Schemas[inSchema.Ref] = &openapi3.SchemaRef{
					Value: inSchema.Value,
				}
			} else {
				// No body fields (e.g. struct{}) — inline empty object schema.
				// This happens for POST/PUT/PATCH where a body is required
				// even when the input struct has no body fields.
				op.RequestBody = &openapi3.RequestBodyRef{
					Value: &openapi3.RequestBody{
						Required: true,
						Content: map[string]*openapi3.MediaType{
							"application/json": {
								Schema: &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: &openapi3.Types{"object"},
									},
								},
							},
						},
					},
				}
			}
		}
	}

	if info != nil {
		op.Summary = info.Summary
		op.Description = info.Description
		op.Tags = info.Tags
	}

	pathItem := a.spec.Paths.Find(path)
	if pathItem == nil {
		pathItem = &openapi3.PathItem{}
		a.spec.Paths.Set(path, pathItem)
	}

	switch method {
	case http.MethodGet:
		pathItem.Get = op
	case http.MethodPost:
		pathItem.Post = op
	case http.MethodPut:
		pathItem.Put = op
	case http.MethodDelete:
		pathItem.Delete = op
	case http.MethodPatch:
		pathItem.Patch = op
	case http.MethodHead:
		pathItem.Head = op
	case http.MethodOptions:
		pathItem.Options = op
	case http.MethodTrace:
		pathItem.Trace = op
	case http.MethodConnect:
		pathItem.Connect = op
	default:
		return fmt.Errorf("method '%s' not supported", method)
	}

	return nil
}

// operationID generates an operation ID like "getItems" or "postUserById" from
// the HTTP method and path.
func operationID(method, path string) string {
	method = strings.ToLower(method)
	segments := strings.Split(strings.Trim(path, "/"), "/")

	var parts []string
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			name := seg[1 : len(seg)-1]
			parts = append(parts, "By"+capitalize(name))
		} else {
			parts = append(parts, capitalize(seg))
		}
	}

	return method + strings.Join(parts, "")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// errorResponseRef creates an OpenAPI response reference for an error component schema.
func errorResponseRef(description, schemaName string) *openapi3.ResponseRef {
	return &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: new(description),
			Content: map[string]*openapi3.MediaType{
				"application/json": {
					Schema: &openapi3.SchemaRef{
						Ref: fmt.Sprintf("#/components/schemas/%s", schemaName),
					},
				},
			},
		},
	}
}

func (a *API) generateSchemaRef(t reflect.Type) (*openapi3.SchemaRef, error) {
	if t == nil {
		return nil, nil
	}
	schema, err := a.specGen.GenerateSchemaRef(t)
	if err != nil {
		return nil, err
	}
	scrubRefs(schema)
	applyRequired(t, schema.Value)
	return schema, nil
}

// generateQueryParams produces OpenAPI parameter definitions for a query struct type.
// Only fields with `query` tags are included.
func (a *API) generateQueryParams(t reflect.Type) ([]*openapi3.ParameterRef, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("query type must be a struct, got %s", t.Kind())
	}

	var params []*openapi3.ParameterRef
	for field := range t.Fields() {
		if !field.IsExported() {
			continue
		}
		if !hasQueryTag(field) {
			continue
		}
		name := queryFieldName(field)
		schema := fieldToOpenAPISchema(field.Type)

		// Apply validation constraints
		if err := validateSchemaCustomizer(name, field.Type, field.Tag, schema.Value); err != nil {
			return nil, err
		}

		required := hasRule(field.Tag.Get("validate"), "required")

		params = append(params, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:     name,
				In:       "query",
				Required: required,
				Schema:   schema,
			},
		})
	}
	return params, nil
}

// generateFormSchema builds an inline OpenAPI schema and encoding map for multipart/form-data.
// Only fields with `form` tags are included; query-tagged fields are skipped.
// The encoding map is populated for fields with `accept` tags.
func generateFormSchema(t reflect.Type) (*openapi3.Schema, map[string]*openapi3.Encoding) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	schema := &openapi3.Schema{
		Type:       &openapi3.Types{"object"},
		Properties: make(openapi3.Schemas),
	}
	var encoding map[string]*openapi3.Encoding

	for field := range t.Fields() {
		if !field.IsExported() || !hasFormTag(field) {
			continue
		}

		name := formFieldName(field)

		var propSchema *openapi3.SchemaRef
		switch field.Type {
		case fileHeaderType:
			// Single file upload
			propSchema = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:   &openapi3.Types{"string"},
					Format: "binary",
				},
			}
		case fileHeaderSliceType:
			// Multiple file upload
			propSchema = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: &openapi3.Types{"array"},
					Items: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type:   &openapi3.Types{"string"},
							Format: "binary",
						},
					},
				},
			}
		default:
			// Text form field
			propSchema = fieldToOpenAPISchema(field.Type)
			_ = validateSchemaCustomizer(name, field.Type, field.Tag, propSchema.Value)
		}

		schema.Properties[name] = propSchema

		// Add encoding entry for fields with accept tags
		if accept := field.Tag.Get("accept"); accept != "" && isFileField(field) {
			if encoding == nil {
				encoding = make(map[string]*openapi3.Encoding)
			}
			encoding[name] = &openapi3.Encoding{
				ContentType: accept,
			}
		}

		if hasRule(field.Tag.Get("validate"), "required") {
			schema.Required = append(schema.Required, name)
		}
	}

	return schema, encoding
}

// fieldToOpenAPISchema maps a Go type to an OpenAPI schema.
func fieldToOpenAPISchema(t reflect.Type) *openapi3.SchemaRef {
	// Unwrap pointer
	if t.Kind() == reflect.Pointer {
		return fieldToOpenAPISchema(t.Elem())
	}

	// Handle slices
	if t.Kind() == reflect.Slice {
		items := scalarToOpenAPISchema(t.Elem())
		return &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type:  &openapi3.Types{"array"},
				Items: items,
			},
		}
	}

	return scalarToOpenAPISchema(t)
}

// scalarToOpenAPISchema maps a scalar Go type to an OpenAPI schema.
func scalarToOpenAPISchema(t reflect.Type) *openapi3.SchemaRef {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}}
	case reflect.Bool:
		return &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"boolean"}}}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}}
	case reflect.Float32, reflect.Float64:
		return &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"number"}}}
	default:
		return &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}}
	}
}

// stripQueryFields removes query-tagged fields from a body schema's Properties and Required.
func stripQueryFields(t reflect.Type, schema *openapi3.Schema) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || schema == nil {
		return
	}
	for f := range t.Fields() {
		if !f.IsExported() || !hasQueryTag(f) {
			continue
		}
		jname := jsonFieldName(f)
		if jname == "" || jname == "-" {
			continue
		}
		delete(schema.Properties, jname)
		// Remove from Required slice
		for j, req := range schema.Required {
			if req == jname {
				schema.Required = append(schema.Required[:j], schema.Required[j+1:]...)
				break
			}
		}
	}
}

// generateHeaderParams produces OpenAPI parameter definitions for a header struct type.
// Only fields with `header` tags are included. Slices are not supported for headers.
func (a *API) generateHeaderParams(t reflect.Type) ([]*openapi3.ParameterRef, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("header type must be a struct, got %s", t.Kind())
	}

	var params []*openapi3.ParameterRef
	for field := range t.Fields() {
		if !field.IsExported() {
			continue
		}
		if !hasHeaderTag(field) {
			continue
		}
		name := http.CanonicalHeaderKey(headerFieldName(field))
		schema := scalarToOpenAPISchema(field.Type)

		// Apply validation constraints
		if err := validateSchemaCustomizer(name, field.Type, field.Tag, schema.Value); err != nil {
			return nil, err
		}

		required := hasRule(field.Tag.Get("validate"), "required")

		params = append(params, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:     name,
				In:       "header",
				Required: required,
				Schema:   schema,
			},
		})
	}
	return params, nil
}

// stripHeaderFields removes header-tagged fields from a body schema's Properties and Required.
func stripHeaderFields(t reflect.Type, schema *openapi3.Schema) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || schema == nil {
		return
	}
	for f := range t.Fields() {
		if !f.IsExported() || !hasHeaderTag(f) {
			continue
		}
		jname := jsonFieldName(f)
		if jname == "" || jname == "-" {
			continue
		}
		delete(schema.Properties, jname)
		// Remove from Required slice
		for j, req := range schema.Required {
			if req == jname {
				schema.Required = append(schema.Required[:j], schema.Required[j+1:]...)
				break
			}
		}
	}
}

// stripPathFields removes path-tagged fields from a body schema's Properties and Required.
func stripPathFields(t reflect.Type, schema *openapi3.Schema) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || schema == nil {
		return
	}
	for f := range t.Fields() {
		if !f.IsExported() || !hasPathTag(f) {
			continue
		}
		jname := jsonFieldName(f)
		if jname == "" || jname == "-" {
			continue
		}
		delete(schema.Properties, jname)
		for j, req := range schema.Required {
			if req == jname {
				schema.Required = append(schema.Required[:j], schema.Required[j+1:]...)
				break
			}
		}
	}
}

func scrubRefs(s *openapi3.SchemaRef) {
	if s == nil || s.Value == nil {
		return
	}
	// Scrub ref on non-object schemas
	if s.Value.Type != nil && !s.Value.Type.Is("object") {
		s.Ref = ""
	}
	// Recurse into array items
	if s.Value.Items != nil {
		scrubRefs(s.Value.Items)
	}
	// Recurse into properties
	for _, p := range s.Value.Properties {
		scrubRefs(p)
	}
}
