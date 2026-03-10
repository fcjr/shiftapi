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
		eventVariants:      s.cfg.eventVariants,
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

func registerSSERoute[In, Event any](
	router Router,
	method string,
	path string,
	fn SSEHandlerFunc[In, Event],
	options ...RouteOption,
) {
	s := prepareRoute[In](router, method, path, false, options)

	// Set content type for the OpenAPI spec. If WithEvents was used,
	// event variants drive schema generation (oneOf + discriminator).
	// Otherwise, use the Event type parameter for a single-type schema.
	s.cfg.contentType = "text/event-stream"
	var sendVariants map[reflect.Type]string
	if len(s.cfg.eventVariants) == 0 {
		s.cfg.responseSchemaType = reflect.TypeFor[Event]()
	} else {
		sendVariants = make(map[reflect.Type]string, len(s.cfg.eventVariants))
		seen := make(map[string]bool, len(s.cfg.eventVariants))
		for _, ev := range s.cfg.eventVariants {
			name := ev.eventName()
			if seen[name] {
				panic(fmt.Sprintf("shiftapi: duplicate event name %q in WithEvents for %s %s", name, method, path))
			}
			seen[name] = true
			sendVariants[ev.eventPayloadType()] = name
		}
	}

	si := s.schemaInput(method, nil, false, false)
	if err := s.api.updateSchema(si); err != nil {
		panic(fmt.Sprintf("shiftapi: schema generation failed for %s %s: %v", method, s.fullPath, err))
	}

	hc := s.handlerCfg(method, false)
	h := adaptSSE(fn, hc, sendVariants)
	s.wrapAndRegister(router, h)
}



// HandleSSE registers a Server-Sent Events handler for the given pattern.
// The handler receives a typed [SSEWriter] for sending events to the client.
// Input parsing, validation, and middleware work identically to [Handle].
//
// The OpenAPI spec automatically uses "text/event-stream" as the response
// content type, with the Event type parameter generating the event schema.
//
//	shiftapi.HandleSSE(api, "GET /events", func(r *http.Request, in struct{}, sse *shiftapi.SSEWriter[Message]) error {
//	    for msg := range messages(r.Context()) {
//	        if err := sse.Send(msg); err != nil {
//	            return err
//	        }
//	    }
//	    return nil
//	})
func HandleSSE[In, Event any](router Router, pattern string, fn SSEHandlerFunc[In, Event], options ...RouteOption) {
	method, path := parsePattern(pattern)
	registerSSERoute(router, method, path, fn, options...)
}



func registerWSRoute[In any](
	router Router,
	method string,
	path string,
	msgs WSMessages[In],
	wsOpts wsRouteConfig,
) {
	// Build a routeConfig from the wsRouteConfig so we can reuse prepareRoute.
	routeOpts := []RouteOption{}
	if wsOpts.info != nil {
		routeOpts = append(routeOpts, WithRouteInfo(*wsOpts.info))
	}
	for _, e := range wsOpts.errors {
		routeOpts = append(routeOpts, routeOptionFunc(func(cfg *routeConfig) {
			cfg.addError(e)
		}))
	}
	if len(wsOpts.middleware) > 0 {
		routeOpts = append(routeOpts, routeOptionFunc(func(cfg *routeConfig) {
			cfg.addMiddleware(wsOpts.middleware)
		}))
	}
	for _, h := range wsOpts.staticRespHeaders {
		routeOpts = append(routeOpts, routeOptionFunc(func(cfg *routeConfig) {
			cfg.addStaticResponseHeader(h)
		}))
	}

	s := prepareRoute[In](router, method, path, false, routeOpts)

	// Extract recv variants from On handlers.
	recvVariants := make([]MessageVariant, len(msgs.cfg.handlers))
	for i, h := range msgs.cfg.handlers {
		recvVariants[i] = rawMessageVariant{name: h.messageName(), payloadType: h.messagePayloadType()}
	}

	// Validate no duplicate message names.
	validateMessageVariants(msgs.cfg.sendVariants, "WSSends", method, path)
	validateMessageVariants(recvVariants, "WSOn", method, path)

	// Build dispatch map for the receive loop.
	dispatch := make(map[string]wsOnHandler, len(msgs.cfg.handlers))
	for _, h := range msgs.cfg.handlers {
		dispatch[h.messageName()] = h
	}

	// Build send variants map for WSSender auto-wrapping.
	var sendVariantMap map[reflect.Type]string
	if len(msgs.cfg.sendVariants) > 0 {
		sendVariantMap = make(map[reflect.Type]string, len(msgs.cfg.sendVariants))
		for _, v := range msgs.cfg.sendVariants {
			sendVariantMap[v.messagePayloadType()] = v.messageName()
		}
	}

	// Build path field map for AsyncAPI channel parameters.
	pathFields := make(map[string]reflect.StructField)
	if s.pathType != nil {
		pt := s.pathType
		for pt.Kind() == reflect.Pointer {
			pt = pt.Elem()
		}
		if pt.Kind() == reflect.Struct {
			for f := range pt.Fields() {
				if f.IsExported() && hasPathTag(f) {
					pathFields[pathFieldName(f)] = f
				}
			}
		}
	}

	// For AsyncAPI, use nil types when variants are present (variants carry the types).
	var sendType, recvType reflect.Type
	if len(msgs.cfg.sendVariants) == 0 {
		sendType = nil
	}
	if len(recvVariants) == 0 {
		recvType = nil
	}

	// Register in AsyncAPI spec.
	if err := s.api.addWSChannel(
		s.fullPath, sendType, recvType,
		msgs.cfg.sendVariants, recvVariants,
		wsOpts.info, pathFields,
	); err != nil {
		panic(fmt.Sprintf("shiftapi: AsyncAPI generation failed for %s %s: %v", method, s.fullPath, err))
	}

	hc := s.handlerCfg(method, false)
	h := adaptWSMessages[In](dispatch, sendVariantMap, hc, wsOpts.wsAcceptOptions)
	s.wrapAndRegister(router, h)
}



// HandleWS registers a WebSocket endpoint for the given pattern. Message
// handling is defined by [WSOn] handlers collected in a [Websocket] block.
// The framework manages the receive loop, dispatching incoming messages
// to the matching handler.
//
// Input parsing, validation, and middleware work identically to [Handle].
// WebSocket endpoints are documented in an AsyncAPI 2.4 spec served at
// GET /asyncapi.json.
//
//	shiftapi.HandleWS(api, "GET /chat",
//	    shiftapi.Websocket[struct{}](
//	        shiftapi.WSOn("message", func(ctx context.Context, ws *shiftapi.WSSender, _ struct{}, m UserMessage) error {
//	            return ws.Send(ctx, ChatMessage{User: "echo", Text: m.Text})
//	        }),
//	        shiftapi.WSSends(
//	            shiftapi.MessageType[ChatMessage]("chat"),
//	        ),
//	    ),
//	)
func HandleWS[In any](router Router, pattern string, msgs WSMessages[In], options ...WSOption) {
	method, path := parsePattern(pattern)
	wsOpts := applyWSOptions(options)
	registerWSRoute(router, method, path, msgs, wsOpts)
}

// rawMessageVariant is a non-generic MessageVariant implementation built
// from On handlers at registration time.
type rawMessageVariant struct {
	name        string
	payloadType reflect.Type
}

func (r rawMessageVariant) messageName() string              { return r.name }
func (r rawMessageVariant) messagePayloadType() reflect.Type { return r.payloadType }

func validateMessageVariants(variants []MessageVariant, optName, method, path string) {
	if len(variants) == 0 {
		return
	}
	seen := make(map[string]bool, len(variants))
	for _, v := range variants {
		name := v.messageName()
		if seen[name] {
			panic(fmt.Sprintf("shiftapi: duplicate message name %q in %s for %s %s", name, optName, method, path))
		}
		seen[name] = true
	}
}

