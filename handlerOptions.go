package shiftapi

import (
	"net/http"
)

type routeConfig struct {
	info              *RouteInfo
	status            int
	errors            []errorEntry
	middleware         []func(http.Handler) http.Handler
	staticRespHeaders []staticResponseHeader
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
