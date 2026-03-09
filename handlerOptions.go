package shiftapi

import (
	"net/http"
	"reflect"
)

type routeConfig struct {
	info               *RouteInfo
	status             int
	errors             []errorEntry
	middleware          []func(http.Handler) http.Handler
	staticRespHeaders  []staticResponseHeader
	contentType        string         // custom response media type
	responseSchemaType reflect.Type   // optional type for schema generation under the content type
	eventVariants      []EventVariant // SSE event variants for oneOf schema generation
	wsOptions      *WSAcceptOptions
	wsSendVariants []MessageVariant
	wsRecvVariants []MessageVariant
}

func (c *routeConfig) addError(e errorEntry) {
	c.errors = append(c.errors, e)
}

func (c *routeConfig) addMiddleware(mw []func(http.Handler) http.Handler) {
	c.middleware = append(c.middleware, mw...)
}

func (c *routeConfig) addStaticResponseHeader(h staticResponseHeader) {
	c.staticRespHeaders = append(c.staticRespHeaders, h)
}

func applyRouteOptions(opts []RouteOption) routeConfig {
	cfg := routeConfig{status: http.StatusOK}
	for _, opt := range opts {
		opt.applyToRoute(&cfg)
	}
	return cfg
}

// RouteInfo provides metadata for a route that appears in the OpenAPI spec
// and the generated documentation UI.
type RouteInfo struct {
	Summary     string
	Description string
	Tags        []string
}

// WithRouteInfo sets the route's OpenAPI metadata (summary, description, tags).
//
//	shiftapi.Handle(api, "POST /greet", greet, shiftapi.WithRouteInfo(shiftapi.RouteInfo{
//	    Summary: "Greet a person",
//	    Tags:    []string{"greetings"},
//	}))
func WithRouteInfo(info RouteInfo) routeOptionFunc {
	return func(cfg *routeConfig) {
		cfg.info = &info
	}
}

// WithStatus sets the success HTTP status code for the route (default: 200).
// Use this for routes that should return 201 Created, 204 No Content, etc.
func WithStatus(status int) routeOptionFunc {
	return func(cfg *routeConfig) {
		cfg.status = status
	}
}

// ResponseSchemaOption carries a type for deferred OpenAPI schema generation
// with [WithContentType].
type ResponseSchemaOption struct {
	typ reflect.Type
}

// ResponseSchema captures the type T for OpenAPI schema generation. The actual
// schema is generated at registration time using the API's configured schema
// customizer, so enum lookups and validation constraints are applied correctly.
func ResponseSchema[T any]() ResponseSchemaOption {
	return ResponseSchemaOption{typ: reflect.TypeFor[T]()}
}

// WithContentType sets a custom response content type for the route's OpenAPI
// spec. An optional [ResponseSchemaOption] produced by [ResponseSchema] can be
// passed to include a schema under the specified media type.
//
// For [HandleRaw] routes, this determines how the response appears in the
// OpenAPI spec. For [Handle] routes, this overrides the default
// "application/json" media type key.
//
//	shiftapi.HandleRaw(api, "GET /events", sseHandler,
//	    shiftapi.WithContentType("text/event-stream"),
//	)
//	shiftapi.HandleRaw(api, "GET /events", sseHandler,
//	    shiftapi.WithContentType("text/event-stream", shiftapi.ResponseSchema[Event]()),
//	)
func WithContentType(contentType string, opts ...ResponseSchemaOption) routeOptionFunc {
	return func(cfg *routeConfig) {
		cfg.contentType = contentType
		if len(opts) > 0 {
			cfg.responseSchemaType = opts[0].typ
		}
	}
}

// WithEvents registers named SSE event types for OpenAPI schema generation.
// Each [EventVariant] maps an event name to a payload type, producing a oneOf
// schema with a discriminator in the OpenAPI spec. The generated TypeScript
// client yields a discriminated union type.
//
// Use with [HandleSSE] and [SSEWriter.SendEvent] to send different payload
// types under different event names:
//
//	shiftapi.HandleSSE(api, "GET /chat", chatHandler,
//	    shiftapi.WithEvents(
//	        shiftapi.EventType[MessageData]("message"),
//	        shiftapi.EventType[JoinData]("join"),
//	    ),
//	)
func WithEvents(variants ...EventVariant) routeOptionFunc {
	return func(cfg *routeConfig) {
		cfg.eventVariants = append(cfg.eventVariants, variants...)
	}
}

// WSAcceptOptions configures the WebSocket upgrade for [HandleWS] routes.
type WSAcceptOptions struct {
	// Subprotocols lists the WebSocket subprotocols to negotiate with the
	// client. The empty subprotocol is always negotiated per RFC 6455.
	Subprotocols []string

	// OriginPatterns lists host patterns for authorized cross-origin requests.
	// The request host is always authorized. Each pattern is matched case
	// insensitively with [path.Match]. Include a URI scheme ("://") to match
	// against "scheme://host".
	//
	// In dev mode (shiftapidev build tag), all origins are allowed by default.
	OriginPatterns []string
}

// WithWSAcceptOptions sets the WebSocket upgrade options for [HandleWS] routes.
// Use this to configure subprotocols, allowed origins, etc.
//
//	shiftapi.HandleWS(api, "GET /ws", handler,
//	    shiftapi.WithWSAcceptOptions(shiftapi.WSAcceptOptions{
//	        Subprotocols:   []string{"graphql-ws"},
//	        OriginPatterns: []string{"example.com"},
//	    }),
//	)
func WithWSAcceptOptions(opts WSAcceptOptions) routeOptionFunc {
	return func(cfg *routeConfig) {
		cfg.wsOptions = &opts
	}
}

// WithSendMessages registers named server-to-client message types for AsyncAPI
// schema generation on a [HandleWS] route. Each [MessageVariant] maps a type
// name to a payload type, producing a oneOf schema with a discriminator on
// the "type" field.
//
// When using WithSendMessages, send messages via [WSConn.SendEvent] which
// wraps the payload in {"type": name, "data": payload}.
//
//	shiftapi.HandleWS(api, "GET /chat", handler,
//	    shiftapi.WithSendMessages(
//	        shiftapi.MessageType[ChatMessage]("chat"),
//	        shiftapi.MessageType[SystemMessage]("system"),
//	    ),
//	)
func WithSendMessages(variants ...MessageVariant) routeOptionFunc {
	return func(cfg *routeConfig) {
		cfg.wsSendVariants = append(cfg.wsSendVariants, variants...)
	}
}

// WithRecvMessages registers named client-to-server message types for AsyncAPI
// schema generation on a [HandleWS] route. Each [MessageVariant] maps a type
// name to a payload type, producing a oneOf schema with a discriminator on
// the "type" field.
//
// When using WithRecvMessages, receive messages via [WSConn.ReceiveEvent]
// which parses the {"type": name, "data": payload} envelope into a
// [WSEvent].
//
//	shiftapi.HandleWS(api, "GET /chat", handler,
//	    shiftapi.WithRecvMessages(
//	        shiftapi.MessageType[UserMessage]("message"),
//	        shiftapi.MessageType[UserCommand]("command"),
//	    ),
//	)
func WithRecvMessages(variants ...MessageVariant) routeOptionFunc {
	return func(cfg *routeConfig) {
		cfg.wsRecvVariants = append(cfg.wsRecvVariants, variants...)
	}
}
