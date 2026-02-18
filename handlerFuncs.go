package shiftapi

import (
	"fmt"
	"net/http"
	"reflect"
)

func registerRoute[Resp any](
	api *API,
	method string,
	path string,
	fn HandlerFunc[Resp],
	options ...RouteOption,
) {
	cfg := applyRouteOptions(options)

	var resp Resp
	outType := reflect.TypeOf(resp)

	if err := api.updateSchema(method, path, nil, outType, cfg.info, cfg.status); err != nil {
		panic(fmt.Sprintf("shiftapi: schema generation failed for %s %s: %v", method, path, err))
	}

	pattern := fmt.Sprintf("%s %s", method, path)
	api.mux.HandleFunc(pattern, adapt(fn, cfg.status))
}

func registerRouteWithBody[Body, Resp any](
	api *API,
	method string,
	path string,
	fn HandlerFuncWithBody[Body, Resp],
	options ...RouteOption,
) {
	cfg := applyRouteOptions(options)

	var body Body
	inType := reflect.TypeOf(body)
	var resp Resp
	outType := reflect.TypeOf(resp)

	if err := api.updateSchema(method, path, inType, outType, cfg.info, cfg.status); err != nil {
		panic(fmt.Sprintf("shiftapi: schema generation failed for %s %s: %v", method, path, err))
	}

	pattern := fmt.Sprintf("%s %s", method, path)
	api.mux.HandleFunc(pattern, adaptWithBody(fn, cfg.status, api.validateBody))
}

// No-body methods

// Get registers a GET handler.
func Get[Resp any](api *API, path string, fn HandlerFunc[Resp], options ...RouteOption) {
	registerRoute(api, http.MethodGet, path, fn, options...)
}

// Delete registers a DELETE handler.
func Delete[Resp any](api *API, path string, fn HandlerFunc[Resp], options ...RouteOption) {
	registerRoute(api, http.MethodDelete, path, fn, options...)
}

// Head registers a HEAD handler.
func Head[Resp any](api *API, path string, fn HandlerFunc[Resp], options ...RouteOption) {
	registerRoute(api, http.MethodHead, path, fn, options...)
}

// Options registers an OPTIONS handler.
func Options[Resp any](api *API, path string, fn HandlerFunc[Resp], options ...RouteOption) {
	registerRoute(api, http.MethodOptions, path, fn, options...)
}

// Trace registers a TRACE handler.
func Trace[Resp any](api *API, path string, fn HandlerFunc[Resp], options ...RouteOption) {
	registerRoute(api, http.MethodTrace, path, fn, options...)
}

// Body methods

// Post registers a POST handler.
func Post[Body, Resp any](api *API, path string, fn HandlerFuncWithBody[Body, Resp], options ...RouteOption) {
	registerRouteWithBody(api, http.MethodPost, path, fn, options...)
}

// Put registers a PUT handler.
func Put[Body, Resp any](api *API, path string, fn HandlerFuncWithBody[Body, Resp], options ...RouteOption) {
	registerRouteWithBody(api, http.MethodPut, path, fn, options...)
}

// Patch registers a PATCH handler.
func Patch[Body, Resp any](api *API, path string, fn HandlerFuncWithBody[Body, Resp], options ...RouteOption) {
	registerRouteWithBody(api, http.MethodPatch, path, fn, options...)
}

// Connect registers a CONNECT handler.
func Connect[Resp any](api *API, path string, fn HandlerFunc[Resp], options ...RouteOption) {
	registerRoute(api, http.MethodConnect, path, fn, options...)
}
