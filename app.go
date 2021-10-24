package shiftapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/julienschmidt/httprouter"
)

type ShiftAPI struct {
	router    *httprouter.Router
	schemaGen *openapi3gen.Generator
	schema    *openapi3.T
}

type SchemaParams struct {
	Title string
}

type Params struct {
	SchemaInfo *SchemaParams
}

func New(params *Params) *ShiftAPI {
	router := httprouter.New()
	t := &openapi3.T{
		OpenAPI: "3.0.3",
		Paths:   make(openapi3.Paths),
		Components: openapi3.Components{
			Schemas: make(openapi3.Schemas),
		},
	}
	if params.SchemaInfo != nil {
		t.Info = &openapi3.Info{
			Title: params.SchemaInfo.Title,
		}
	}

	app := &ShiftAPI{
		router:    router,
		schemaGen: openapi3gen.NewGenerator(),
		schema:    t,
	}
	router.GET("/openapi.json", app.serveSchema)
	router.GET("/docs", app.serveDocs)

	return app
}

func (s *ShiftAPI) serveSchema(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
	b, err := s.schema.MarshalJSON()
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	_, _ = res.Write(b)
}

func (s *ShiftAPI) serveDocs(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
	title := ""
	if s.schema.Info != nil {
		title = s.schema.Info.Title
	}
	err := genRedocHTML(redocData{
		Title:   title,
		SpecURL: "http://localhost:8080/openapi.json",
	}, res)
	if err != nil {
		fmt.Println(err)
		res.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *ShiftAPI) Serve() error {
	return http.ListenAndServe(":8080", s.router)
}

type HandlerOpts struct {
	Summary     string
	Description string
}

func (s *ShiftAPI) POST(path string, handler interface{}, status int, opts *HandlerOpts) error {
	return s.handle(http.MethodPost, path, handler, status, opts)
}

func (s *ShiftAPI) handle(method, path string, handler interface{}, status int, opts *HandlerOpts) error {
	hv := reflect.ValueOf(handler)
	if hv.Kind() != reflect.Func {
		return fmt.Errorf("invalid handler func")
	}

	f := hv.Type()
	numIn := f.NumIn()
	if numIn != 1 {
		return fmt.Errorf("handlers must have 1 struct ptr argument")
	}
	inType := f.In(0)
	if inType.Kind() != reflect.Ptr || inType.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("handlers must have 1 struct ptr argument")
	}

	numOut := f.NumOut()
	if numOut != 2 {
		return fmt.Errorf("handlers must have 2 outputs, a struct ptr an a *shiftapi.Error")
	}
	outType := f.Out(0)
	if outType.Kind() != reflect.Ptr || outType.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("handlers first output must be a struct ptr")
	}
	errType := f.Out(1)
	if errType.Kind() != reflect.Ptr || errType.Elem() != reflect.ValueOf(Error{}).Type() {
		return fmt.Errorf("handlers second output must be of type *shiftapi.Error")
	}

	if err := s.updateSchema(method, path, f, inType.Elem(), outType.Elem(), status, opts); err != nil {
		return err
	}

	innerHandler := func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		dec := json.NewDecoder(req.Body)
		inPtr := reflect.New(inType)
		err := dec.Decode(inPtr.Interface())
		if err != nil {
			fmt.Println(err)
			res.WriteHeader(400)
			return
		}

		out := hv.Call([]reflect.Value{inPtr.Elem()})
		if outErr := out[1].Interface().(*Error); outErr != nil {
			res.WriteHeader(outErr.Code)
			_, _ = res.Write(outErr.Body)
			return
		}
		res.WriteHeader(status)
		res.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(res).Encode(out[0].Interface())
	}

	s.router.Handle(method, path, innerHandler)
	return nil
}
