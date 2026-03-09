package shiftapi

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

func registerRoute[In, Resp any](
	router Router,
	method string,
	path string,
	fn HandlerFunc[In, Resp],
	options ...RouteOption,
) {
	rd := router.routerImpl()
	api := rd.api
	fullPath := strings.TrimRight(rd.prefix, "/") + path

	cfg := applyRouteOptions(options)

	var in In
	inType := reflect.TypeOf(in)
	// Dereference pointer to get the underlying struct type
	rawInType := inType
	for rawInType != nil && rawInType.Kind() == reflect.Pointer {
		rawInType = rawInType.Elem()
	}

	hasQuery, hasBody, hasForm := partitionFields(rawInType)

	var queryType reflect.Type
	if hasQuery {
		queryType = rawInType
	}
	// POST/PUT/PATCH conventionally carry a request body, so always attempt
	// body decode for these methods — even when the input is struct{}.
	// This means Post(api, path, func(r, _ struct{}) ...) requires at least "{}".
	methodRequiresBody := method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch
	decodeBody := !hasForm && (hasBody || methodRequiresBody)

	var bodyType reflect.Type
	if !hasForm {
		if hasBody {
			bodyType = inType
		} else if methodRequiresBody {
			bodyType = rawInType
		}
	}

	var resp Resp
	outType := reflect.TypeOf(resp)

	// Merge: rd.errors already contains API globals + group errors.
	// Append route-level errors on top.
	allErrors := append(rd.errors, cfg.errors...)

	if err := api.updateSchema(method, fullPath, queryType, bodyType, outType, hasForm, rawInType, cfg.info, cfg.status, allErrors); err != nil {
		panic(fmt.Sprintf("shiftapi: schema generation failed for %s %s: %v", method, fullPath, err))
	}

	errLookup := buildErrorLookup(allErrors)

	pattern := fmt.Sprintf("%s %s", method, fullPath)
	var h http.Handler = adapt(fn, cfg.status, api.validateBody, hasQuery, decodeBody, hasForm, api.maxUploadSize, errLookup, api.badRequestFn, api.internalServerFn)
	// Apply route-level middleware (innermost), then group-level (outermost).
	// Reverse order so the first middleware in the slice wraps outermost.
	for i := len(cfg.middleware) - 1; i >= 0; i-- {
		h = cfg.middleware[i](h)
	}
	for i := len(rd.middleware) - 1; i >= 0; i-- {
		h = rd.middleware[i](h)
	}
	api.mux.Handle(pattern, h)
	devNotifyRoute(api)
}

// Get registers a handler for GET requests at the given path. The path
// follows [net/http.ServeMux] patterns, including wildcards like /users/{id}.
// Path parameters are accessible via [http.Request.PathValue].
func Get[In, Resp any](router Router, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(router, http.MethodGet, path, fn, options...)
}

// Post registers a handler for POST requests at the given path. The request
// body is automatically decoded from JSON (or multipart/form-data if the In
// type has form-tagged fields). Validation is applied before the handler runs.
func Post[In, Resp any](router Router, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(router, http.MethodPost, path, fn, options...)
}

// Put registers a PUT handler.
func Put[In, Resp any](router Router, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(router, http.MethodPut, path, fn, options...)
}

// Patch registers a PATCH handler.
func Patch[In, Resp any](router Router, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(router, http.MethodPatch, path, fn, options...)
}

// Delete registers a DELETE handler.
func Delete[In, Resp any](router Router, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(router, http.MethodDelete, path, fn, options...)
}

// Head registers a HEAD handler.
func Head[In, Resp any](router Router, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(router, http.MethodHead, path, fn, options...)
}

// Options registers an OPTIONS handler.
func Options[In, Resp any](router Router, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(router, http.MethodOptions, path, fn, options...)
}

// Trace registers a TRACE handler.
func Trace[In, Resp any](router Router, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(router, http.MethodTrace, path, fn, options...)
}

// Connect registers a CONNECT handler.
func Connect[In, Resp any](router Router, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(router, http.MethodConnect, path, fn, options...)
}
