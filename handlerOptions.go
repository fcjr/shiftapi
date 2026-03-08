package shiftapi

import (
	"net/http"
	"reflect"
)

// RouteOption configures a route registered with [Get], [Post], [Put], etc.
type RouteOption func(*routeConfig)

type routeConfig struct {
	info   *RouteInfo
	status int
	errors []errorEntry
}

type errorEntry struct {
	status int
	typ    reflect.Type // always pointer type for errors.As
}

// errorLookup maps concrete error types to their HTTP status codes.
// Built once at route registration time for O(1) lookups during error handling.
type errorLookup map[reflect.Type]int

func buildErrorLookup(entries []errorEntry) errorLookup {
	if len(entries) == 0 {
		return nil
	}
	lookup := make(errorLookup, len(entries)*2)
	for _, e := range entries {
		lookup[e.typ] = e.status           // *T
		lookup[e.typ.Elem()] = e.status    // T (for value-receiver errors)
	}
	return lookup
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

// WithError declares that an error of type T may be returned at the given HTTP
// status code. T must implement [error] and its struct fields are reflected into
// the OpenAPI schema. At runtime, if a handler returns an error matching T (via
// [errors.As]), it is serialized as JSON with the declared status code.
//
// Use [WithGlobalError] to register an error type on all routes instead.
//
//	shiftapi.Post(api, "/users", createUser,
//	    shiftapi.WithError[*ConflictError](http.StatusConflict),
//	)
func WithError[T error](status int) RouteOption {
	t := reflect.TypeFor[T]()
	// Normalize to pointer so errors.As works correctly.
	if t.Kind() != reflect.Pointer {
		t = reflect.PointerTo(t)
	}
	return func(cfg *routeConfig) {
		cfg.errors = append(cfg.errors, errorEntry{
			status: status,
			typ:    t,
		})
	}
}
