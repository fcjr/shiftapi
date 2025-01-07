package shiftapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
)

type Handler interface {
	register(server *ShiftAPI) error

	// unimplementable is a method that should never be called.
	// It is simply used to ensure that the Handler interface can only be implemented
	// internally by the shiftapi package.
	unimplementable()
}

type HandlerOption interface {
	// unimplementable is a method that should never be called.
	// It is simply used to ensure that the HandlerOption interface can only be implemented
	// internally by the shiftapi package.
	unimplementable()
}

type ValidBody any // TODO can we type constrain to a struct?

type HandlerFunc[RequestBody ValidBody, ResponseBody ValidBody] func(
	ctx context.Context,
	headers http.Header,
	requestBody RequestBody,
) (responseBody ResponseBody, err error)

// TODO pass status code
type handler[RequestBody ValidBody, ResponseBody ValidBody] struct {
	method      string
	path        string
	handlerFunc HandlerFunc[RequestBody, ResponseBody]
	options     []func(HandlerFunc[RequestBody, ResponseBody]) HandlerFunc[RequestBody, ResponseBody]
}

// ensure handler implements Handler at compile time
var _ Handler = handler[any, any]{}

func (h handler[RequestBody, ResponseBody]) unimplementable() {
	panic("unimplementable called")
}

func (h handler[RequestBody, ResponseBody]) register(server *ShiftAPI) error {
	if err := h.updateSchema(server); err != nil {
		return err
	}

	pattern := fmt.Sprintf("%s %s", h.method, h.path)
	stdHandler := h.stdHandler(server.baseContext)
	server.mux.HandleFunc(pattern, stdHandler)
	return nil
}

func (h handler[RequestBody, ResponseBody]) updateSchema(server *ShiftAPI) error {

	var in RequestBody
	inType := reflect.TypeOf(in)
	inSchema, err := server.generateSchemaRef(inType)
	if err != nil {
		return err
	}

	var out ResponseBody
	outType := reflect.TypeOf(out)
	outSchema, err := server.generateSchemaRef(outType)
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

	var oPath *openapi3.PathItem
	switch h.method {
	case http.MethodPost:
		oPath = &openapi3.PathItem{
			Post: &openapi3.Operation{
				// Summary:     opts.Summary,
				RequestBody: requestBody,
				// Description: opts.Description,
				Responses: responses,
			},
		}
	}
	if oPath == nil {
		return fmt.Errorf("method '%s' not implemented", h.method)
	}
	server.spec.Paths.Set(h.path, oPath)
	// server.spec.Components.Responses.Default()

	server.spec.Components.Schemas[inSchema.Ref] = &openapi3.SchemaRef{
		Value: inSchema.Value,
	}
	server.spec.Components.Schemas[outSchema.Ref] = &openapi3.SchemaRef{
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

func (h handler[RequestBody, ResponseBody]) stdHandler(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: valdiate request body?
		var requestBody RequestBody
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		responseBody, err := h.handlerFunc(
			ctx,
			r.Header,
			requestBody,
		)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(responseBody); err != nil {
			http.Error(w, "error encoding response", http.StatusInternalServerError)
			return
		}
	}
}
