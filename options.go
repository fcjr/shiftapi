package shiftapi

import (
	"net/http"
	"reflect"
)

// sharedConfig is the common interface implemented by [*API], [*groupConfig],
// and [*routeConfig]. It provides the operations that are meaningful at all
// three levels: adding errors, middleware, and static response headers.
type sharedConfig interface {
	addError(errorEntry)
	addMiddleware([]func(http.Handler) http.Handler)
	addStaticResponseHeader(staticResponseHeader)
}

// staticResponseHeader is a fixed name/value pair set on every response.
type staticResponseHeader struct {
	name  string
	value string
}

// Option is the primary option type. It works at all levels: [New],
// [API.Group]/[Group.Group], and route registration functions ([Get], [Post],
// etc.). Options are composable via [ComposeOptions].
type Option func(sharedConfig)

func (f Option) applyToAPI(api *API)           { f(api) }
func (f Option) applyToGroup(cfg *groupConfig) { f(cfg) }
func (f Option) applyToRoute(cfg *routeConfig) { f(cfg) }

// APIOption configures an [API] created with [New]. Both [Option] and
// API-specific options (like [WithInfo]) implement this interface.
type APIOption interface {
	applyToAPI(*API)
}

// GroupOption configures a [Group] created with [API.Group] or [Group.Group].
// [Option] implements this interface.
type GroupOption interface {
	applyToGroup(*groupConfig)
}

// RouteOption configures a route registered with [Get], [Post], [Put], etc.
// Both [Option] and route-specific options (like [WithStatus]) implement
// this interface.
type RouteOption interface {
	applyToRoute(*routeConfig)
}

// apiOptionFunc is a function that implements [APIOption].
type apiOptionFunc func(*API)

func (f apiOptionFunc) applyToAPI(api *API) { f(api) }

// groupOptionFunc is a function that implements [GroupOption].
type groupOptionFunc func(*groupConfig)

func (f groupOptionFunc) applyToGroup(cfg *groupConfig) { f(cfg) }

// routeOptionFunc is a function that implements [RouteOption].
type routeOptionFunc func(*routeConfig)

func (f routeOptionFunc) applyToRoute(cfg *routeConfig) { f(cfg) }

// errorEntry maps an error type to an HTTP status code.
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
		lookup[e.typ] = e.status        // *T
		lookup[e.typ.Elem()] = e.status // T (for value-receiver errors)
	}
	return lookup
}

// WithError declares that an error of type T may be returned at the given HTTP
// status code. T must implement [error] and its struct fields are reflected into
// the OpenAPI schema. At runtime, if a handler returns an error matching T (via
// [errors.As]), it is serialized as JSON with the declared status code.
//
// WithError returns an [Option] that works at any level:
//   - [New] — applies to all routes (API-level)
//   - [API.Group] / [Group.Group] — applies to all routes in the group
//   - [Handle] — applies to a single route
//
//	api := shiftapi.New(
//	    shiftapi.WithError[*AuthError](http.StatusUnauthorized),
//	)
//	v1 := api.Group("/api/v1",
//	    shiftapi.WithError[*RateLimitError](http.StatusTooManyRequests),
//	)
//	shiftapi.Handle(v1, "GET /users/{id}", getUser,
//	    shiftapi.WithError[*NotFoundError](http.StatusNotFound),
//	)
func WithError[T error](status int) Option {
	t := reflect.TypeFor[T]()
	// Normalize to pointer so errors.As works correctly.
	if t.Kind() != reflect.Pointer {
		t = reflect.PointerTo(t)
	}
	return func(c sharedConfig) {
		c.addError(errorEntry{status: status, typ: t})
	}
}

// WithMiddleware applies standard HTTP middleware. Middleware functions are
// applied in order: the first argument wraps outermost.
//
// WithMiddleware returns an [Option] that works at any level:
//   - [New] — applies to all routes (API-level)
//   - [API.Group] / [Group.Group] — applies to all routes in the group
//   - [Handle] — applies to a single route
//
//	api := shiftapi.New(
//	    shiftapi.WithMiddleware(cors, logging),
//	)
//	v1 := api.Group("/api/v1",
//	    shiftapi.WithMiddleware(auth),
//	)
//	shiftapi.Handle(v1, "GET /admin", getAdmin,
//	    shiftapi.WithMiddleware(adminOnly),
//	)
func WithMiddleware(mw ...func(http.Handler) http.Handler) Option {
	return func(c sharedConfig) {
		c.addMiddleware(mw)
	}
}

// WithResponseHeader sets a static response header on every response. The
// header is also documented in the OpenAPI spec for each affected route.
//
// WithResponseHeader returns an [Option] that works at any level:
//   - [New] — applies to all routes (API-level)
//   - [API.Group] / [Group.Group] — applies to all routes in the group
//   - [Handle] — applies to a single route
//
// Static headers are applied in API → Group → Route order. If the same header
// name is declared at multiple levels, the later level wins. Dynamic headers
// (header struct tags on the response type) are applied after static headers
// and take precedence for the same name.
//
//	api := shiftapi.New(
//	    shiftapi.WithResponseHeader("X-Content-Type-Options", "nosniff"),
//	)
//	v1 := api.Group("/api/v1",
//	    shiftapi.WithResponseHeader("X-API-Version", "1"),
//	)
//	shiftapi.Handle(v1, "GET /users", listUsers,
//	    shiftapi.WithResponseHeader("Cache-Control", "max-age=3600"),
//	)
func WithResponseHeader(name, value string) Option {
	return func(c sharedConfig) {
		c.addStaticResponseHeader(staticResponseHeader{name: http.CanonicalHeaderKey(name), value: value})
	}
}

// ComposeOptions combines multiple [Option] values into a single [Option].
// Use this to create reusable option bundles that work at any level.
//
//	func WithAuth() shiftapi.Option {
//	    return shiftapi.ComposeOptions(
//	        shiftapi.WithMiddleware(authMiddleware),
//	        shiftapi.WithError[*AuthError](http.StatusUnauthorized),
//	    )
//	}
func ComposeOptions(opts ...Option) Option {
	return func(c sharedConfig) {
		for _, opt := range opts {
			opt(c)
		}
	}
}

// ComposeAPIOptions combines multiple [APIOption] values into a single [APIOption].
// Since [Option] implements [APIOption], both shared and API-specific options
// can be mixed.
func ComposeAPIOptions(opts ...APIOption) APIOption {
	return apiOptionFunc(func(api *API) {
		for _, opt := range opts {
			opt.applyToAPI(api)
		}
	})
}

// ComposeGroupOptions combines multiple [GroupOption] values into a single
// [GroupOption]. Since [Option] implements [GroupOption], both shared and
// group-specific options can be mixed.
func ComposeGroupOptions(opts ...GroupOption) GroupOption {
	return groupOptionFunc(func(cfg *groupConfig) {
		for _, opt := range opts {
			opt.applyToGroup(cfg)
		}
	})
}

// ComposeRouteOptions combines multiple [RouteOption] values into a single
// [RouteOption]. Since [Option] implements [RouteOption], both shared and
// route-specific options can be mixed.
//
//	createOpts := shiftapi.ComposeRouteOptions(
//	    shiftapi.WithStatus(http.StatusCreated),
//	    shiftapi.WithError[*ConflictError](http.StatusConflict),
//	)
func ComposeRouteOptions(opts ...RouteOption) RouteOption {
	return routeOptionFunc(func(cfg *routeConfig) {
		for _, opt := range opts {
			opt.applyToRoute(cfg)
		}
	})
}
