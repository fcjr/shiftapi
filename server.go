package shiftapi

import (
	"encoding/json"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/go-playground/validator/v10"
)

// API collects typed handler registrations, generates an OpenAPI schema,
// and implements http.Handler so it can be used with any standard server.
type API struct {
	spec     *openapi3.T
	specGen  *openapi3gen.Generator
	mux      *http.ServeMux
	validate *validator.Validate
}

// New creates a new API with the given options.
func New(options ...Option) *API {
	api := &API{
		spec: &openapi3.T{
			OpenAPI: "3.1",
			Paths:   &openapi3.Paths{},
			Components: &openapi3.Components{
				Schemas: make(openapi3.Schemas),
			},
		},
		specGen: openapi3gen.NewGenerator(
			openapi3gen.SchemaCustomizer(validateSchemaCustomizer),
		),
		mux:      http.NewServeMux(),
		validate: validator.New(),
	}
	for _, opt := range options {
		opt(api)
	}
	api.mux.HandleFunc("GET /openapi.json", api.serveSpec)
	api.mux.HandleFunc("GET /docs", api.serveDocs)
	api.mux.HandleFunc("GET /", api.redirectTo("/docs"))
	return api
}

// ServeHTTP implements http.Handler.
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

func (a *API) validateBody(val any) error {
	return validateStruct(a.validate, val)
}

func (a *API) serveSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(a.spec); err != nil {
		http.Error(w, "error encoding spec", http.StatusInternalServerError)
	}
}

func (a *API) serveDocs(w http.ResponseWriter, r *http.Request) {
	title := ""
	if a.spec.Info != nil {
		title = a.spec.Info.Title
	}
	if err := genDocsHTML(docsData{
		Title:   title,
		SpecURL: "/openapi.json",
	}, w); err != nil {
		http.Error(w, "error generating docs", http.StatusInternalServerError)
	}
}

func (a *API) redirectTo(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, path, http.StatusTemporaryRedirect)
	}
}
