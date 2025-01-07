package shiftapi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

type ShiftAPI struct {
	baseContext context.Context
	spec        *openapi3.T
	specGen     *openapi3gen.Generator
	mux         *http.ServeMux
}

func New(
	ctx context.Context,
	options ...func(*ShiftAPI) *ShiftAPI,
) *ShiftAPI {
	mux := http.NewServeMux()
	spec := &openapi3.T{
		OpenAPI: "3.1",
		Paths:   &openapi3.Paths{},
		Components: &openapi3.Components{
			Schemas: make(openapi3.Schemas),
		},
	}

	api := &ShiftAPI{
		baseContext: ctx,
		spec:        spec,
		specGen:     openapi3gen.NewGenerator(),
		mux:         mux,
	}
	for _, option := range options {
		api = option(api)
	}
	mux.HandleFunc("GET /openapi.json", api.serveSchema)
	mux.HandleFunc("GET /docs", api.serveDocs)
	mux.HandleFunc("GET /", api.redirectTo("/docs"))
	return api
}

// Register adds 1 or more handlers to the server.
// The handlers are expected to be created via the shiftapi.Post, shiftapi.Get,
// shiftapi.Put, shiftapi.Patch, and shiftapi.Delete functions.
func (s *ShiftAPI) Register(handlers ...Handler) {
	for _, h := range handlers {
		h.register(s)
	}
}

func (s *ShiftAPI) redirectTo(path string) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		http.Redirect(res, req, path, http.StatusTemporaryRedirect)
	}
}

func (s *ShiftAPI) serveSchema(res http.ResponseWriter, req *http.Request) {
	b, err := s.spec.MarshalJSON()
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	_, _ = res.Write(b)
}

func (s *ShiftAPI) serveDocs(res http.ResponseWriter, req *http.Request) {
	title := ""
	if s.spec.Info != nil {
		title = s.spec.Info.Title
	}
	err := genRedocHTML(redocData{
		Title:   title,
		SpecURL: "/openapi.json",
	}, res)
	if err != nil {
		fmt.Println(err)
		res.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *ShiftAPI) ListenAndServe(addr string) error {
	// TODO add address to schema & create server separately http server
	// maybe also pass ctx
	return http.ListenAndServe(addr, s.mux)
}
