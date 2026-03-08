package shiftapi

import (
	"fmt"
	"net/http"
	"reflect"
)

func registerRoute[In, Resp any](
	api *API,
	method string,
	path string,
	fn HandlerFunc[In, Resp],
	options ...RouteOption,
) {
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

	// Merge API-level errors with route-level errors.
	allErrors := append(api.globalErrors, cfg.errors...)

	if err := api.updateSchema(method, path, queryType, bodyType, outType, hasForm, rawInType, cfg.info, cfg.status, allErrors); err != nil {
		panic(fmt.Sprintf("shiftapi: schema generation failed for %s %s: %v", method, path, err))
	}

	errLookup := buildErrorLookup(allErrors)

	pattern := fmt.Sprintf("%s %s", method, path)
	api.mux.HandleFunc(pattern, adapt(fn, cfg.status, api.validateBody, hasQuery, decodeBody, hasForm, api.maxUploadSize, errLookup, api.badRequestFn, api.internalServerFn))
}

// Get registers a handler for GET requests at the given path. The path
// follows [net/http.ServeMux] patterns, including wildcards like /users/{id}.
// Path parameters are accessible via [http.Request.PathValue].
func Get[In, Resp any](api *API, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(api, http.MethodGet, path, fn, options...)
}

// Post registers a handler for POST requests at the given path. The request
// body is automatically decoded from JSON (or multipart/form-data if the In
// type has form-tagged fields). Validation is applied before the handler runs.
func Post[In, Resp any](api *API, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(api, http.MethodPost, path, fn, options...)
}

// Put registers a PUT handler.
func Put[In, Resp any](api *API, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(api, http.MethodPut, path, fn, options...)
}

// Patch registers a PATCH handler.
func Patch[In, Resp any](api *API, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(api, http.MethodPatch, path, fn, options...)
}

// Delete registers a DELETE handler.
func Delete[In, Resp any](api *API, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(api, http.MethodDelete, path, fn, options...)
}

// Head registers a HEAD handler.
func Head[In, Resp any](api *API, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(api, http.MethodHead, path, fn, options...)
}

// Options registers an OPTIONS handler.
func Options[In, Resp any](api *API, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(api, http.MethodOptions, path, fn, options...)
}

// Trace registers a TRACE handler.
func Trace[In, Resp any](api *API, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(api, http.MethodTrace, path, fn, options...)
}

// Connect registers a CONNECT handler.
func Connect[In, Resp any](api *API, path string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	registerRoute(api, http.MethodConnect, path, fn, options...)
}
