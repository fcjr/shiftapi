package shiftapi

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
)

func (s *ShiftAPI) updateSchema(method, path string, inType, outType reflect.Type) error {

	inSchema, err := s.generateSchemaRef(inType)
	if err != nil {
		return err
	}

	outSchema, err := s.generateSchemaRef(outType)
	if err != nil {
		return err
	}
	responses := openapi3.NewResponses()
	responseContent := make(map[string]*openapi3.MediaType)
	responseContent["application/json"] = &openapi3.MediaType{
		Schema: &openapi3.SchemaRef{
			Ref: fmt.Sprintf("#/components/schemas/%s", outSchema.Ref),
		},
	}
	responses.Set("200", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: String("Success"),
			Content:     responseContent,
		},
	})

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

	pathItem := s.spec.Paths.Find(path)
	if pathItem == nil {
		pathItem = &openapi3.PathItem{}
		s.spec.Paths.Set(path, pathItem)
	}

	op := &openapi3.Operation{
		// Summary:     opts.Summary,
		RequestBody: requestBody,
		// Description: opts.Description,
		Responses: responses,
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
	default:
		return fmt.Errorf("method '%s' not supported", method)
	}
	// s.spec.Components.Responses.Default()

	s.spec.Components.Schemas[inSchema.Ref] = &openapi3.SchemaRef{
		Value: inSchema.Value,
	}
	s.spec.Components.Schemas[outSchema.Ref] = &openapi3.SchemaRef{
		Value: outSchema.Value,
	}
	return nil
}

func (s *ShiftAPI) generateSchemaRef(t reflect.Type) (*openapi3.SchemaRef, error) {
	schema, err := s.specGen.GenerateSchemaRef(t)
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
		if !p.Value.Type.Is("object") {
			p.Ref = ""
		}
	}
}
