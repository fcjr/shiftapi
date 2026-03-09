package shiftapi

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

type routeConfig struct {
	info              *RouteInfo
	status            int
	errors            []errorEntry
	middleware         []func(http.Handler) http.Handler
	staticRespHeaders []staticResponseHeader
	contentType       string              // custom response media type
	responseSchema    *openapi3.SchemaRef // optional schema for the content type
}

func (c *routeConfig) addError(e errorEntry) {
	c.errors = append(c.errors, e)
}

func (c *routeConfig) addMiddleware(mw []func(http.Handler) http.Handler) {
	c.middleware = append(c.middleware, mw...)
}

func (c *routeConfig) addStaticResponseHeader(h staticResponseHeader) {
	c.staticRespHeaders = append(c.staticRespHeaders, h)
}

func applyRouteOptions(opts []RouteOption) routeConfig {
	cfg := routeConfig{status: http.StatusOK}
	for _, opt := range opts {
		opt.applyToRoute(&cfg)
	}
	return cfg
}

// RouteInfo provides metadata for a route that appears in the OpenAPI spec
// and the generated documentation UI.
type RouteInfo struct {
	Summary     string
	Description string
	Tags        []string
}

// WithRouteInfo sets the route's OpenAPI metadata (summary, description, tags).
//
//	shiftapi.Handle(api, "POST /greet", greet, shiftapi.WithRouteInfo(shiftapi.RouteInfo{
//	    Summary: "Greet a person",
//	    Tags:    []string{"greetings"},
//	}))
func WithRouteInfo(info RouteInfo) routeOptionFunc {
	return func(cfg *routeConfig) {
		cfg.info = &info
	}
}

// WithStatus sets the success HTTP status code for the route (default: 200).
// Use this for routes that should return 201 Created, 204 No Content, etc.
func WithStatus(status int) routeOptionFunc {
	return func(cfg *routeConfig) {
		cfg.status = status
	}
}

// ResponseSchemaOption carries an OpenAPI schema for use with [WithContentType].
type ResponseSchemaOption struct {
	schema *openapi3.SchemaRef
}

// ResponseSchema generates an OpenAPI schema from T for use as the optional
// second argument to [WithContentType]. T is reflected into a schema using
// the same logic as typed handler responses.
func ResponseSchema[T any]() ResponseSchemaOption {
	t := reflect.TypeFor[T]()
	gen := openapi3gen.NewGenerator()
	schema, err := gen.GenerateSchemaRef(t)
	if err != nil {
		panic(fmt.Sprintf("shiftapi: failed to generate response schema for %s: %v", t, err))
	}
	scrubRefs(schema)
	applyRequired(t, schema.Value)
	return ResponseSchemaOption{schema: schema}
}

// WithContentType sets a custom response content type for the route's OpenAPI
// spec. An optional [ResponseSchemaOption] produced by [ResponseSchema] can be
// passed to include a schema under the specified media type.
//
// For [HandleRaw] routes, this determines how the response appears in the
// OpenAPI spec. For [Handle] routes, this overrides the default
// "application/json" media type key.
//
//	shiftapi.HandleRaw(api, "GET /events", sseHandler,
//	    shiftapi.WithContentType("text/event-stream"),
//	)
//	shiftapi.HandleRaw(api, "GET /events", sseHandler,
//	    shiftapi.WithContentType("text/event-stream", shiftapi.ResponseSchema[Event]()),
//	)
func WithContentType(contentType string, opts ...ResponseSchemaOption) routeOptionFunc {
	return func(cfg *routeConfig) {
		cfg.contentType = contentType
		if len(opts) > 0 {
			cfg.responseSchema = opts[0].schema
		}
	}
}
