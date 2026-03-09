package shiftapi

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// validMethods is the set of HTTP methods recognized by Handle.
var validMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodPost:    true,
	http.MethodPut:     true,
	http.MethodPatch:   true,
	http.MethodDelete:  true,
	http.MethodConnect: true,
	http.MethodOptions: true,
	http.MethodTrace:   true,
}

// parsePattern splits a pattern like "GET /users/{id}" into method and path.
// It panics if the pattern is malformed or uses an unknown HTTP method.
func parsePattern(pattern string) (method, path string) {
	method, path, ok := strings.Cut(pattern, " ")
	if !ok || method == "" || path == "" {
		panic(fmt.Sprintf("shiftapi: invalid pattern %q — must be \"METHOD /path\"", pattern))
	}
	if !validMethods[method] {
		panic(fmt.Sprintf("shiftapi: unknown HTTP method %q in pattern %q", method, pattern))
	}
	return method, path
}

// routeSetup holds the computed values from input type reflection that are
// shared between registerRoute and registerRawRoute.
type routeSetup struct {
	api             *API
	cfg             routeConfig
	fullPath        string
	inType          reflect.Type
	rawInType       reflect.Type
	hasPath         bool
	hasQuery        bool
	hasHeader       bool
	hasBody         bool
	hasForm         bool
	queryType       reflect.Type
	headerType      reflect.Type
	pathType        reflect.Type
	bodyType        reflect.Type
	allErrors       []errorEntry
	allStaticHeaders []staticResponseHeader
	errLookup       errorLookup
	muxPattern      string
}

// prepareRoute performs the input type reflection, path validation, and schema
// setup shared by both registerRoute and registerRawRoute. The decodeBody and
// bodyType fields depend on whether the caller forces body decode for
// POST/PUT/PATCH (Handle does, HandleRaw does not), so they are computed here
// based on the forceMethodBody flag.
func prepareRoute[In any](router Router, method, path string, forceMethodBody bool, options []RouteOption) routeSetup {
	rd := router.routerImpl()
	api := rd.api
	fullPath := strings.TrimRight(rd.prefix, "/") + path

	cfg := applyRouteOptions(options)

	var in In
	inType := reflect.TypeOf(in)
	rawInType := inType
	for rawInType != nil && rawInType.Kind() == reflect.Pointer {
		rawInType = rawInType.Elem()
	}

	hasPath, hasQuery, hasHeader, hasBody, hasForm := partitionFields(rawInType)

	if hasPath {
		matches := pathParamRe.FindAllStringSubmatch(fullPath, -1)
		routeParams := make(map[string]bool, len(matches))
		for _, m := range matches {
			routeParams[m[1]] = true
		}
		validatePathFields(rawInType, routeParams)
	}

	var queryType reflect.Type
	if hasQuery {
		queryType = rawInType
	}
	var headerType reflect.Type
	if hasHeader {
		headerType = rawInType
	}

	methodRequiresBody := forceMethodBody && (method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch)

	var bodyType reflect.Type
	if !hasForm {
		if hasBody {
			bodyType = inType
		} else if methodRequiresBody {
			bodyType = rawInType
		}
	}

	allErrors := append(rd.errors, cfg.errors...)
	allStaticHeaders := append(rd.staticRespHeaders, cfg.staticRespHeaders...)

	var pathType reflect.Type
	if hasPath {
		pathType = rawInType
	}

	errLookup := buildErrorLookup(allErrors)
	muxPattern := fmt.Sprintf("%s %s", method, fullPath)

	// Apply middleware chain: route-level (innermost), then group-level (outermost).
	// Stored in cfg/rd for wrapHandler to apply.

	return routeSetup{
		api:              api,
		cfg:              cfg,
		fullPath:         fullPath,
		inType:           inType,
		rawInType:        rawInType,
		hasPath:          hasPath,
		hasQuery:         hasQuery,
		hasHeader:        hasHeader,
		hasBody:          hasBody,
		hasForm:          hasForm,
		queryType:        queryType,
		headerType:       headerType,
		pathType:         pathType,
		bodyType:         bodyType,
		allErrors:        allErrors,
		allStaticHeaders: allStaticHeaders,
		errLookup:        errLookup,
		muxPattern:       muxPattern,
	}
}

// schemaInput builds the parameters for updateSchema from the route setup.
func (s *routeSetup) schemaInput(method string, outType reflect.Type, hasRespHeader, noBody bool) schemaInput {
	return schemaInput{
		method:             method,
		path:               s.fullPath,
		pathType:           s.pathType,
		queryType:          s.queryType,
		headerType:         s.headerType,
		bodyType:           s.bodyType,
		outType:            outType,
		hasRespHeader:      hasRespHeader,
		noBody:             noBody,
		hasForm:            s.hasForm,
		formType:           s.rawInType,
		info:               s.cfg.info,
		status:             s.cfg.status,
		errors:             s.allErrors,
		staticHeaders:      s.allStaticHeaders,
		contentType:        s.cfg.contentType,
		responseSchemaType: s.cfg.responseSchemaType,
	}
}

// wrapAndRegister applies middleware and registers the handler on the mux.
func (s *routeSetup) wrapAndRegister(router Router, h http.Handler) {
	rd := router.routerImpl()
	for i := len(s.cfg.middleware) - 1; i >= 0; i-- {
		h = s.cfg.middleware[i](h)
	}
	for i := len(rd.middleware) - 1; i >= 0; i-- {
		h = rd.middleware[i](h)
	}
	s.api.mux.Handle(s.muxPattern, h)
}

// handlerCfg builds the per-request handler configuration from the route setup
// and API. The forceMethodBody flag controls whether POST/PUT/PATCH methods
// force body decode even without json-tagged fields (true for Handle, false for
// HandleRaw).
func (s *routeSetup) handlerCfg(method string, forceMethodBody bool) *handlerConfig {
	decodeBody := s.hasBody
	if !decodeBody && !s.hasForm && forceMethodBody {
		decodeBody = method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch
	}
	if s.hasForm {
		decodeBody = false
	}
	return &handlerConfig{
		hasPath:          s.hasPath,
		hasQuery:         s.hasQuery,
		hasHeader:        s.hasHeader,
		decodeBody:       decodeBody,
		hasForm:          s.hasForm,
		maxUploadSize:    s.api.maxUploadSize,
		staticHeaders:    s.allStaticHeaders,
		errLookup:        s.errLookup,
		validate:         s.api.validateBody,
		badRequestFn:     s.api.badRequestFn,
		internalServerFn: s.api.internalServerFn,
	}
}

func registerRoute[In, Resp any](
	router Router,
	method string,
	path string,
	fn HandlerFunc[In, Resp],
	options ...RouteOption,
) {
	s := prepareRoute[In](router, method, path, true, options)

	var resp Resp
	outType := reflect.TypeOf(resp)
	hasRespHeader := hasRespHeaderFields(outType)

	var respEnc *respEncoder
	if hasRespHeader {
		respEnc = newRespEncoder(outType)
	}

	noBody := isNoBodyStatus(s.cfg.status)

	// Panic if a no-body status code is used with a response type that has JSON body fields.
	if noBody && outType != nil {
		ot := outType
		for ot.Kind() == reflect.Pointer {
			ot = ot.Elem()
		}
		if ot.Kind() == reflect.Struct {
			for f := range ot.Fields() {
				if f.IsExported() && !hasHeaderTag(f) {
					panic(fmt.Sprintf("shiftapi: status %d must not have a response body; response type %s has JSON body field %q — use struct{} or a header-only struct", s.cfg.status, ot.Name(), f.Name))
				}
			}
		}
	}

	si := s.schemaInput(method, outType, hasRespHeader, noBody)
	if err := s.api.updateSchema(si); err != nil {
		panic(fmt.Sprintf("shiftapi: schema generation failed for %s %s: %v", method, s.fullPath, err))
	}

	hc := s.handlerCfg(method, true)
	h := adapt(fn, hc, s.cfg.status, noBody, respEnc)
	s.wrapAndRegister(router, h)
}

// Handle registers a typed handler for the given pattern. The pattern follows
// [net/http.ServeMux] conventions: "METHOD /path", e.g. "GET /users/{id}".
//
// Path parameters can be declared on the input struct with path:"name" tags
// for automatic parsing and validation, or accessed via [http.Request.PathValue].
//
// For POST, PUT, and PATCH methods, the request body is automatically decoded
// from JSON (or multipart/form-data if the In type has form-tagged fields).
// Validation is applied before the handler runs.
//
//	shiftapi.Handle(api, "GET /users/{id}", getUser)
//	shiftapi.Handle(api, "POST /users", createUser)
//	shiftapi.Handle(api, "DELETE /items/{id}", deleteItem,
//	    shiftapi.WithStatus(http.StatusNoContent),
//	)
func Handle[In, Resp any](router Router, pattern string, fn HandlerFunc[In, Resp], options ...RouteOption) {
	method, path := parsePattern(pattern)
	registerRoute(router, method, path, fn, options...)
}

func registerRawRoute[In any](
	router Router,
	method string,
	path string,
	fn RawHandlerFunc[In],
	options ...RouteOption,
) {
	s := prepareRoute[In](router, method, path, false, options)

	si := s.schemaInput(method, nil, false, false)
	if err := s.api.updateSchema(si); err != nil {
		panic(fmt.Sprintf("shiftapi: schema generation failed for %s %s: %v", method, s.fullPath, err))
	}

	hc := s.handlerCfg(method, false)
	h := adaptRaw(fn, hc)
	s.wrapAndRegister(router, h)
}

// HandleRaw registers a raw handler for the given pattern. Unlike [Handle],
// the handler receives the [http.ResponseWriter] directly and is responsible
// for writing the response. Input parsing, validation, and middleware work
// identically to [Handle].
//
// Use HandleRaw for responses that cannot be expressed as a typed struct:
// Server-Sent Events, file downloads, WebSocket upgrades, etc.
//
//	shiftapi.HandleRaw(api, "GET /events", sseHandler,
//	    shiftapi.WithContentType("text/event-stream"),
//	)
func HandleRaw[In any](router Router, pattern string, fn RawHandlerFunc[In], options ...RouteOption) {
	method, path := parsePattern(pattern)
	registerRawRoute(router, method, path, fn, options...)
}

