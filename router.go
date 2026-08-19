package shiftapi

import "net/http"

// routeTarget is the common interface for route registration. Both [*API] and
// [*Group] implement it, letting the internal registration helpers accept
// either.
type routeTarget interface {
	routerImpl() routerData
}

type routerData struct {
	api               *API
	prefix            string
	errors            []errorEntry                      // accumulated errors from API globals + group chain
	middleware        []func(http.Handler) http.Handler // accumulated middleware from group chain
	staticRespHeaders []staticResponseHeader            // accumulated static response headers from group chain
}
