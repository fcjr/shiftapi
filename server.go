package shiftapi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

type ShiftAPI struct {
	baseContext context.Context
	spec        *v3.Document
	mux         *http.ServeMux
}

func New(
	ctx context.Context,
	options ...func(*ShiftAPI) *ShiftAPI,
) *ShiftAPI {
	mux := http.NewServeMux()
	spec := &v3.Document{
		Version: "3.1",
		Paths: &v3.Paths{
			PathItems: orderedmap.New[string, *v3.PathItem](),
		},
		Components: &v3.Components{
			Schemas:         orderedmap.New[string, *base.SchemaProxy](),
			Responses:       orderedmap.New[string, *v3.Response](),
			Parameters:      orderedmap.New[string, *v3.Parameter](),
			Examples:        orderedmap.New[string, *base.Example](),
			RequestBodies:   orderedmap.New[string, *v3.RequestBody](),
			Headers:         orderedmap.New[string, *v3.Header](),
			SecuritySchemes: orderedmap.New[string, *v3.SecurityScheme](),
			Links:           orderedmap.New[string, *v3.Link](),
			Callbacks:       orderedmap.New[string, *v3.Callback](),
			PathItems:       orderedmap.New[string, *v3.PathItem](),
		},
	}

	api := &ShiftAPI{
		baseContext: ctx,
		spec:        spec,
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
	b, err := s.spec.RenderJSON("  ")
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
