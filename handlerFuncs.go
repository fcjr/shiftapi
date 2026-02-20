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

	if err := api.updateSchema(method, path, nil, nil, outType, cfg.info, cfg.status); err != nil {
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

	if err := api.updateSchema(method, path, nil, inType, outType, cfg.info, cfg.status); err != nil {
		panic(fmt.Sprintf("shiftapi: schema generation failed for %s %s: %v", method, path, err))
	}

	pattern := fmt.Sprintf("%s %s", method, path)
	api.mux.HandleFunc(pattern, adaptWithBody(fn, cfg.status, api.validateBody))
}

func registerRouteWithQuery[Query, Resp any](
	api *API,
	method string,
	path string,
	fn HandlerFuncWithQuery[Query, Resp],
	options ...RouteOption,
) {
	cfg := applyRouteOptions(options)

	var query Query
	queryType := reflect.TypeOf(query)
	var resp Resp
	outType := reflect.TypeOf(resp)

	if err := api.updateSchema(method, path, queryType, nil, outType, cfg.info, cfg.status); err != nil {
		panic(fmt.Sprintf("shiftapi: schema generation failed for %s %s: %v", method, path, err))
	}

	pattern := fmt.Sprintf("%s %s", method, path)
	api.mux.HandleFunc(pattern, adaptWithQuery(fn, cfg.status, api.validateBody))
}

func registerRouteWithQueryAndBody[Query, Body, Resp any](
	api *API,
	method string,
	path string,
	fn HandlerFuncWithQueryAndBody[Query, Body, Resp],
	options ...RouteOption,
) {
	cfg := applyRouteOptions(options)

	var query Query
	queryType := reflect.TypeOf(query)
	var body Body
	inType := reflect.TypeOf(body)
	var resp Resp
	outType := reflect.TypeOf(resp)

	if err := api.updateSchema(method, path, queryType, inType, outType, cfg.info, cfg.status); err != nil {
		panic(fmt.Sprintf("shiftapi: schema generation failed for %s %s: %v", method, path, err))
	}

	pattern := fmt.Sprintf("%s %s", method, path)
	api.mux.HandleFunc(pattern, adaptWithQueryAndBody(fn, cfg.status, api.validateBody))
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

// Query methods (no body)

// GetWithQuery registers a GET handler with typed query parameters.
func GetWithQuery[Query, Resp any](api *API, path string, fn HandlerFuncWithQuery[Query, Resp], options ...RouteOption) {
	registerRouteWithQuery(api, http.MethodGet, path, fn, options...)
}

// DeleteWithQuery registers a DELETE handler with typed query parameters.
func DeleteWithQuery[Query, Resp any](api *API, path string, fn HandlerFuncWithQuery[Query, Resp], options ...RouteOption) {
	registerRouteWithQuery(api, http.MethodDelete, path, fn, options...)
}

// HeadWithQuery registers a HEAD handler with typed query parameters.
func HeadWithQuery[Query, Resp any](api *API, path string, fn HandlerFuncWithQuery[Query, Resp], options ...RouteOption) {
	registerRouteWithQuery(api, http.MethodHead, path, fn, options...)
}

// Query + body methods

// PostWithQuery registers a POST handler with typed query parameters and a request body.
func PostWithQuery[Query, Body, Resp any](api *API, path string, fn HandlerFuncWithQueryAndBody[Query, Body, Resp], options ...RouteOption) {
	registerRouteWithQueryAndBody(api, http.MethodPost, path, fn, options...)
}

// PutWithQuery registers a PUT handler with typed query parameters and a request body.
func PutWithQuery[Query, Body, Resp any](api *API, path string, fn HandlerFuncWithQueryAndBody[Query, Body, Resp], options ...RouteOption) {
	registerRouteWithQueryAndBody(api, http.MethodPut, path, fn, options...)
}

// PatchWithQuery registers a PATCH handler with typed query parameters and a request body.
func PatchWithQuery[Query, Body, Resp any](api *API, path string, fn HandlerFuncWithQueryAndBody[Query, Body, Resp], options ...RouteOption) {
	registerRouteWithQueryAndBody(api, http.MethodPatch, path, fn, options...)
}
