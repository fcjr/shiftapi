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

func (a *API) updateSchema(method, path string, inType, outType reflect.Type, info *RouteInfo, status int) error {
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

func scrubRefs(s *openapi3.SchemaRef) {
	if len(s.Value.Properties) == 0 {
		return
	}
	for _, p := range s.Value.Properties {
		if !p.Value.Type.Is("object") {
			p.Ref = ""
		}
	}
}
