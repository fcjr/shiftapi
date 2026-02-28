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

func (a *API) updateSchema(method, path string, queryType, headerType, inType, outType reflect.Type, info *RouteInfo, status int) error {
	op := &openapi3.Operation{
		OperationID: operationID(method, path),
		Responses:   openapi3.NewResponses(),
	}

	// Path parameters
	for _, match := range pathParamRe.FindAllStringSubmatch(path, -1) {
		op.Parameters = append(op.Parameters, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:     match[1],
				In:       "path",
				Required: true,
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"string"},
					},
				},
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
	outSchema, err := a.generateSchemaRef(outType)
	if err != nil {
		return err
	}
	if outSchema != nil {
		content := make(map[string]*openapi3.MediaType)
		content["application/json"] = &openapi3.MediaType{
			Schema: &openapi3.SchemaRef{
				Ref: fmt.Sprintf("#/components/schemas/%s", outSchema.Ref),
			},
		}
		op.Responses.Set(statusStr, &openapi3.ResponseRef{
			Value: &openapi3.Response{
				Description: stringPtr(http.StatusText(status)),
				Content:     content,
			},
		})
		a.spec.Components.Schemas[outSchema.Ref] = &openapi3.SchemaRef{
			Value: outSchema.Value,
		}
	}

	// Default error response
	op.Responses.Set("default", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: stringPtr("Error"),
			Content: map[string]*openapi3.MediaType{
				"application/json": {
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"object"},
							Properties: openapi3.Schemas{
								"message": &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: &openapi3.Types{"string"},
									},
								},
								"errors": &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: &openapi3.Types{"array"},
										Items: &openapi3.SchemaRef{
											Value: &openapi3.Schema{
												Type: &openapi3.Types{"object"},
												Properties: openapi3.Schemas{
													"field": &openapi3.SchemaRef{
														Value: &openapi3.Schema{
															Type: &openapi3.Types{"string"},
														},
													},
													"message": &openapi3.SchemaRef{
														Value: &openapi3.Schema{
															Type: &openapi3.Types{"string"},
														},
													},
												},
												Required: []string{"field", "message"},
											},
										},
									},
								},
							},
							Required: []string{"message"},
						},
					},
				},
			},
		},
	})

	// Request body schema (only for methods with bodies)
	if inType != nil {
		inSchema, err := a.generateSchemaRef(inType)
		if err != nil {
			return err
		}
		if inSchema != nil {
			// Strip query-tagged and header-tagged fields from the body schema
			stripQueryFields(inType, inSchema.Value)
			stripHeaderFields(inType, inSchema.Value)

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
	for i := range t.NumField() {
		field := t.Field(i)
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
	for i := range t.NumField() {
		f := t.Field(i)
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
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		if !hasHeaderTag(field) {
			continue
		}
		name := headerFieldName(field)
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
	for i := range t.NumField() {
		f := t.Field(i)
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

func scrubRefs(s *openapi3.SchemaRef) {
	if s == nil || s.Value == nil || len(s.Value.Properties) == 0 {
		return
	}
	for _, p := range s.Value.Properties {
		if p == nil || p.Value == nil {
			continue
		}
		if !p.Value.Type.Is("object") {
			p.Ref = ""
		}
	}
}
