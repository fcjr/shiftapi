package shiftapi

import (
	"net/http"
	"slices"
)

// Group is a sub-router that registers routes under a common path prefix
// with shared error types and middleware. Create one with [API.Group] or
// nest with [Group.Group].
type Group struct {
	api               *API
	prefix            string
	errors            []errorEntry
	middleware        []func(http.Handler) http.Handler
	staticRespHeaders []staticResponseHeader
}

func (g *Group) routerImpl() routerData {
	return routerData{
		api:               g.api,
		prefix:            g.prefix,
		errors:            g.errors,
		middleware:        g.middleware,
		staticRespHeaders: g.staticRespHeaders,
	}
}

// Group creates a sub-router with the given path prefix and options.
// Routes registered on the returned Group are prefixed with the given path.
// Error types and middleware registered via options apply to all routes in the
// group.
//
//	v1 := api.Group("/api/v1",
//	    shiftapi.WithError[*RateLimitError](http.StatusTooManyRequests),
//	    shiftapi.WithMiddleware(auth, logging),
//	)
//	shiftapi.Handle(v1, "GET /users", listUsers)  // registers GET /api/v1/users
func (a *API) Group(prefix string, opts ...GroupOption) *Group {
	var cfg groupConfig
	for _, opt := range opts {
		opt.applyToGroup(&cfg)
	}
	return &Group{
		api:               a,
		prefix:            prefix,
		errors:            append(slices.Clone(a.globalErrors), cfg.errors...),
		middleware:        append(slices.Clone(a.middleware), cfg.middleware...),
		staticRespHeaders: append(slices.Clone(a.staticRespHeaders), cfg.staticRespHeaders...),
	}
}

// Group creates a nested sub-router. The prefix is appended to the parent
// group's prefix, and error types and middleware are inherited from the parent.
func (g *Group) Group(prefix string, opts ...GroupOption) *Group {
	var cfg groupConfig
	for _, opt := range opts {
		opt.applyToGroup(&cfg)
	}
	return &Group{
		api:               g.api,
		prefix:            g.prefix + prefix,
		errors:            append(slices.Clone(g.errors), cfg.errors...),
		middleware:        append(slices.Clone(g.middleware), cfg.middleware...),
		staticRespHeaders: append(slices.Clone(g.staticRespHeaders), cfg.staticRespHeaders...),
	}
}

type groupConfig struct {
	errors            []errorEntry
	middleware        []func(http.Handler) http.Handler
	staticRespHeaders []staticResponseHeader
}

func (c *groupConfig) addError(e errorEntry) {
	c.errors = append(c.errors, e)
}

func (c *groupConfig) addMiddleware(mw []func(http.Handler) http.Handler) {
	c.middleware = append(c.middleware, mw...)
}

func (c *groupConfig) addStaticResponseHeader(h staticResponseHeader) {
	c.staticRespHeaders = append(c.staticRespHeaders, h)
}
