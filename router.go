package shiftapi

import "net/http"

// Router is the common interface for registering routes. Both [*API] and
// [*Group] implement Router, so [Get], [Post], [Put], etc. accept either.
type Router interface {
	routerImpl() routerData
}

type routerData struct {
	api               *API
	prefix            string
	errors            []errorEntry                      // accumulated errors from API globals + group chain
	middleware        []func(http.Handler) http.Handler // accumulated middleware from group chain
	staticRespHeaders []staticResponseHeader            // accumulated static response headers from group chain
}
