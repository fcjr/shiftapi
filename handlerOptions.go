package shiftapi

import "net/http"

// RouteOption configures a route.
type RouteOption func(*routeConfig)

type routeConfig struct {
	info   *RouteInfo
	status int
}

func applyRouteOptions(opts []RouteOption) routeConfig {
	cfg := routeConfig{status: http.StatusOK}
	for _, opt := range opts {
		opt(&cfg)
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
//	shiftapi.Post(api, "/greet", greet, shiftapi.WithRouteInfo(shiftapi.RouteInfo{
//	    Summary: "Greet a person",
//	    Tags:    []string{"greetings"},
//	}))
func WithRouteInfo(info RouteInfo) RouteOption {
	return func(cfg *routeConfig) {
		cfg.info = &info
	}
}

// WithStatus sets the success HTTP status code for the route (default: 200).
// Use this for routes that should return 201 Created, 204 No Content, etc.
func WithStatus(status int) RouteOption {
	return func(cfg *routeConfig) {
		cfg.status = status
	}
}
