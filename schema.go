package shiftapi

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/fcjr/shiftapi/internal/utils"
	"github.com/getkin/kin-openapi/openapi3"
)

func (s *ShiftAPI) updateSchema(method, path string, handlerFunc, in, out reflect.Type, status int, opts *HandlerOpts) error {

	inSchema, err := s.generateSchemaRef(in)
	if err != nil {
		return err
	}
	outSchema, err := s.generateSchemaRef(out)
	if err != nil {
		return err
	}
	responses := make(openapi3.Responses)
	responseContent := make(map[string]*openapi3.MediaType)
	responseContent["application/json"] = &openapi3.MediaType{
		Schema: &openapi3.SchemaRef{
			Ref: fmt.Sprintf("#/components/schemas/%s", outSchema.Ref),
		},
	}
	responses[fmt.Sprint(status)] = &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: utils.String("Success"),
			Content:     responseContent,
		},
	}

	requestContent := make(map[string]*openapi3.MediaType)
	requestContent["application/json"] = &openapi3.MediaType{
		Schema: &openapi3.SchemaRef{
			Ref: fmt.Sprintf("#/components/schemas/%s", inSchema.Ref),
		},
	}
	requestBody := &openapi3.RequestBodyRef{
		Value: &openapi3.RequestBody{
			Content: requestContent,
		},
	}

	var oPath *openapi3.PathItem
	switch method {
	case http.MethodPost:
		oPath = &openapi3.PathItem{
			Post: &openapi3.Operation{
				Summary:     opts.Summary,
				RequestBody: requestBody,
				Description: opts.Description,
				Responses:   responses,
			},
		}
	}
	if oPath == nil {
		return fmt.Errorf("method '%s' not implemented", method)
	}
	s.schema.Paths[path] = oPath
	s.schema.Components.Responses.Default()

	s.schema.Components.Schemas[inSchema.Ref] = &openapi3.SchemaRef{
		Value: inSchema.Value,
	}
	s.schema.Components.Schemas[outSchema.Ref] = &openapi3.SchemaRef{
		Value: outSchema.Value,
	}
	return nil
}

func (s *ShiftAPI) generateSchemaRef(t reflect.Type) (*openapi3.SchemaRef, error) {
	schema, err := s.schemaGen.GenerateSchemaRef(t)
	if err != nil {
		return nil, err
	}

	// TODO why tf does kin set ref values for basic types
	scrubRefs(schema)

	return schema, nil
}

func scrubRefs(s *openapi3.SchemaRef) {
	if s.Value.Properties == nil || len(s.Value.Properties) <= 0 {
		return
	}
	for _, p := range s.Value.Properties {
		if p.Value.Type != "object" {
			p.Ref = ""
		}
	}
}
