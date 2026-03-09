package shiftapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/go-playground/validator/v10"
	spec "github.com/swaggest/go-asyncapi/spec-2.4.0"
)

// API is the central type that collects typed handler registrations, generates
// an OpenAPI 3.1 schema, and implements [http.Handler]. Create one with [New]
// and register routes with [Get], [Post], [Put], [Patch], [Delete], etc.
//
// API automatically serves the OpenAPI spec at GET /openapi.json and
// interactive documentation at GET /docs.
type API struct {
	spec             *openapi3.T
	asyncSpec        *spec.AsyncAPI
	specGen          *openapi3gen.Generator
	mux              *http.ServeMux
	validate         *validator.Validate
	maxUploadSize    int64
	badRequestFn     func(error) any // builds the 400 response body from a parse error
	internalServerFn func(error) any // builds the 500 response body from an unmatched error
	enumRegistry          map[reflect.Type][]any            // enum values registered via WithEnum
	globalErrors          []errorEntry                      // error types registered at the API level via WithError
	middleware            []func(http.Handler) http.Handler // middleware registered at the API level via WithMiddleware
	staticRespHeaders     []staticResponseHeader            // static response headers registered at the API level
}

// New creates a new API with the given options. By default the API uses a
// 32 MB upload limit and the standard [github.com/go-playground/validator/v10]
// instance. Use [WithInfo], [WithMaxUploadSize], [WithValidator], and
// [WithExternalDocs] to customize behavior.
func New(options ...APIOption) *API {
	api := &API{
		spec: &openapi3.T{
			OpenAPI: "3.1",
			Paths:   &openapi3.Paths{},
			Components: &openapi3.Components{
				Schemas: make(openapi3.Schemas),
			},
		},
		asyncSpec: &spec.AsyncAPI{
			DefaultContentType: "application/json",
		},
		mux:           http.NewServeMux(),
		validate:      validator.New(),
		maxUploadSize: 32 << 20, // 32 MB
		enumRegistry:  make(map[reflect.Type][]any),
	}
	for _, opt := range options {
		opt.applyToAPI(api)
	}
	api.specGen = openapi3gen.NewGenerator(
		openapi3gen.SchemaCustomizer(api.schemaCustomizer),
	)

	// Set defaults for error response functions if not customized.
	if api.badRequestFn == nil {
		api.badRequestFn = func(_ error) any {
			return &defaultMessage{Message: "bad request"}
		}
		api.spec.Components.Schemas["BadRequestError"] = messageOnlySchemaRef()
	}
	if api.internalServerFn == nil {
		api.internalServerFn = func(_ error) any {
			return &defaultMessage{Message: "internal server error"}
		}
		api.spec.Components.Schemas["InternalServerError"] = messageOnlySchemaRef()
	}
	api.spec.Components.Schemas["ValidationError"] = &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type: &openapi3.Types{"object"},
			Properties: openapi3.Schemas{
				"message": &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &openapi3.Types{"string"}},
				},
				"errors": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"array"},
						Items: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: &openapi3.Types{"object"},
								Properties: openapi3.Schemas{
									"field": &openapi3.SchemaRef{
										Value: &openapi3.Schema{Type: &openapi3.Types{"string"}},
									},
									"message": &openapi3.SchemaRef{
										Value: &openapi3.Schema{Type: &openapi3.Types{"string"}},
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
	}

	// Copy API info to AsyncAPI spec.
	if api.spec.Info != nil {
		api.asyncSpec.Info.Title = api.spec.Info.Title
		api.asyncSpec.Info.Version = api.spec.Info.Version
		api.asyncSpec.Info.Description = api.spec.Info.Description
	}

	api.mux.HandleFunc("GET /openapi.json", api.serveSpec)
	api.mux.HandleFunc("GET /asyncapi.json", api.serveAsyncSpec)
	api.mux.HandleFunc("GET /docs", api.serveDocs)
	api.mux.HandleFunc("GET /docs/ws", api.serveAsyncDocs)
	api.mux.HandleFunc("GET /", api.redirectTo("/docs"))
	return api
}

func (a *API) addError(e errorEntry) {
	a.globalErrors = append(a.globalErrors, e)
}

func (a *API) addMiddleware(mw []func(http.Handler) http.Handler) {
	a.middleware = append(a.middleware, mw...)
}

func (a *API) addStaticResponseHeader(h staticResponseHeader) {
	a.staticRespHeaders = append(a.staticRespHeaders, h)
}

func (a *API) routerImpl() routerData {
	return routerData{
		api:               a,
		prefix:            "",
		errors:            a.globalErrors,
		middleware:         a.middleware,
		staticRespHeaders: a.staticRespHeaders,
	}
}

// ServeHTTP implements http.Handler.
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

func (a *API) validateBody(val any) error {
	return validateStruct(a.validate, val)
}

func (a *API) serveSpec(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(a.spec); err != nil {
		http.Error(w, "error encoding spec", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
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

func (a *API) serveAsyncDocs(w http.ResponseWriter, r *http.Request) {
	if len(a.asyncSpec.Channels) == 0 {
		http.NotFound(w, r)
		return
	}
	title := ""
	if a.spec.Info != nil {
		title = a.spec.Info.Title + " — WebSockets"
	}
	if err := genAsyncDocsHTML(docsData{
		Title:   title,
		SpecURL: "/asyncapi.json",
	}, w); err != nil {
		http.Error(w, "error generating docs", http.StatusInternalServerError)
	}
}

func (a *API) redirectTo(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, path, http.StatusTemporaryRedirect)
	}
}

// defaultMessage is the simple JSON body used for default 400 and 500 responses.
type defaultMessage struct {
	Message string `json:"message"`
}

// messageOnlySchemaRef returns a new schema with a single "message" string property.
func messageOnlySchemaRef() *openapi3.SchemaRef {
	return &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type: &openapi3.Types{"object"},
			Properties: openapi3.Schemas{
				"message": &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &openapi3.Types{"string"}},
				},
			},
			Required: []string{"message"},
		},
	}
}
