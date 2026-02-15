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

// RouteInfo provides metadata for a route in the OpenAPI spec.
type RouteInfo struct {
	Summary     string
	Description string
	Tags        []string
}

// WithRouteInfo sets the route's OpenAPI metadata.
func WithRouteInfo(info RouteInfo) RouteOption {
	return func(cfg *routeConfig) {
		cfg.info = &info
	}
}

// WithStatus sets the success HTTP status code for the route (default: 200).
func WithStatus(status int) RouteOption {
	return func(cfg *routeConfig) {
		cfg.status = status
	}
}
