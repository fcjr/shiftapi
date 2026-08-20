package shiftapi

import (
	"net/http"
	"reflect"
)

type routeConfig struct {
	info               *RouteInfo
	status             int
	errors             []errorEntry
	middleware         []func(http.Handler) http.Handler
	staticRespHeaders  []staticResponseHeader
	contentType        string            // custom response media type
	responseSchemaType reflect.Type      // optional type for schema generation under the content type
	eventVariants      []SSEEventVariant // SSE event variants, set by registerSSERoute
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

func applyHandleOptions(opts []HandleOption) routeConfig {
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

// routeOption implements [RouteOption]: it carries one apply function per
// route kind so a single option value works on Handle, HandleSSE, and
// HandleWS routes.
type routeOption struct {
	routeFn func(*routeConfig)
	wsFn    func(*wsRouteConfig)
	sseFn   func(*sseRouteConfig)
}

func (o routeOption) applyToRoute(cfg *routeConfig)  { o.routeFn(cfg) }
func (o routeOption) applyToWS(cfg *wsRouteConfig)   { o.wsFn(cfg) }
func (o routeOption) applyToSSE(cfg *sseRouteConfig) { o.sseFn(cfg) }

// WithRouteInfo sets the route's OpenAPI metadata (summary, description, tags).
//
//	api.Handle("POST /greet", greet, shiftapi.WithRouteInfo(shiftapi.RouteInfo{
//	    Summary: "Greet a person",
//	    Tags:    []string{"greetings"},
//	}))
func WithRouteInfo(info RouteInfo) RouteOption {
	return routeOption{
		routeFn: func(cfg *routeConfig) { cfg.info = &info },
		wsFn:    func(cfg *wsRouteConfig) { cfg.info = &info },
		sseFn:   func(cfg *sseRouteConfig) { cfg.info = &info },
	}
}

// WithStatus sets the success HTTP status code for the route (default: 200).
// Use this for routes that should return 201 Created, 204 No Content, etc.
func WithStatus(status int) HandleOption {
	return handleOptionFunc(func(cfg *routeConfig) {
		cfg.status = status
	})
}

// ResponseSchemaOption carries a type for deferred OpenAPI schema generation
// with [WithContentType].
type ResponseSchemaOption struct {
	typ reflect.Type
}

// ResponseSchema captures the type T for OpenAPI schema generation. The actual
// schema is generated at registration time using the API's configured schema
// customizer, so enum lookups and validation constraints are applied correctly.
func ResponseSchema[T any]() ResponseSchemaOption {
	return ResponseSchemaOption{typ: reflect.TypeFor[T]()}
}

// WithContentType sets a custom response content type for the route's OpenAPI
// spec. An optional [ResponseSchemaOption] produced by [ResponseSchema] can be
// passed to include a schema under the specified media type.
//
// For [API.HandleRaw] routes, this determines how the response appears in the
// OpenAPI spec. For [API.Handle] routes, this overrides the default
// "application/json" media type key.
//
//	api.HandleRaw("GET /events", sseHandler,
//	    shiftapi.WithContentType("text/event-stream"),
//	)
//	api.HandleRaw("GET /events", sseHandler,
//	    shiftapi.WithContentType("text/event-stream", shiftapi.ResponseSchema[Event]()),
//	)
func WithContentType(contentType string, opts ...ResponseSchemaOption) HandleOption {
	return handleOptionFunc(func(cfg *routeConfig) {
		cfg.contentType = contentType
		if len(opts) > 0 {
			cfg.responseSchemaType = opts[0].typ
		}
	})
}
