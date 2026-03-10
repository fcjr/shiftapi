package shiftapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// WSSender is the send-only WebSocket connection passed to [WSOn] message
// handlers. It provides [WSSender.Send] for writing messages and
// [WSSender.Close] for closing the connection.
//
// [WSSender.Send] automatically wraps the value in a discriminated
// {"type", "data"} envelope based on the concrete Go type registered
// via [WSSends].
type WSSender struct {
	conn         *websocket.Conn
	sendVariants map[reflect.Type]string // nil = raw mode
}

// Send writes a JSON-encoded message to the WebSocket connection. The value
// is automatically wrapped in a {"type": name, "data": value} envelope based
// on its concrete Go type, using the types registered via [WSSends].
func (ws *WSSender) Send(ctx context.Context, v any) error {
	name, ok := ws.sendVariants[reflect.TypeOf(v)]
	if !ok {
		return fmt.Errorf("shiftapi: unregistered send type %T; register with WSSends", v)
	}
	envelope := wsEnvelope[any]{Type: name, Data: v}
	return wsjson.Write(ctx, ws.conn, envelope)
}

// Close closes the WebSocket connection with the given status code and reason.
func (ws *WSSender) Close(status WSStatusCode, reason string) error {
	return ws.conn.Close(websocket.StatusCode(status), reason)
}

// Conn returns the underlying [websocket.Conn] for advanced use cases
// such as binary frames, ping/pong, or custom close handling.
func (ws *WSSender) Conn() *websocket.Conn {
	return ws.conn
}

// WSStatusCode represents a WebSocket close status code as defined in
// RFC 6455 section 7.4.
type WSStatusCode int

// Standard WebSocket close status codes.
const (
	WSStatusNormalClosure   WSStatusCode = 1000
	WSStatusGoingAway       WSStatusCode = 1001
	WSStatusProtocolError   WSStatusCode = 1002
	WSStatusUnsupportedData WSStatusCode = 1003
	WSStatusInternalError   WSStatusCode = 1011
)

// WSCloseStatus extracts the WebSocket close status code from an error.
// Returns -1 if the error is nil or not a WebSocket close error.
func WSCloseStatus(err error) WSStatusCode {
	return WSStatusCode(websocket.CloseStatus(err))
}

// wsEnvelope is the wire format for discriminated WebSocket messages.
type wsEnvelope[T any] struct {
	Type string `json:"type"`
	Data T      `json:"data"`
}

// wsEvent represents a received discriminated WebSocket message with raw
// data for deferred decoding.
type wsEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// WebsocketHandler configures a WebSocket endpoint. Both [WSOn] message
// handlers and [WSSends] type registrations implement this interface.
// Pass them to [Websocket] to build a complete endpoint.
type WebsocketHandler interface {
	applyToWebsocket(*websocketConfig)
}

// websocketConfig holds the configuration built by [Websocket] from
// [WebsocketHandler] values.
type websocketConfig struct {
	handlers     []wsOnHandler
	sendVariants []MessageVariant
}

// wsOnHandler is the internal interface for a typed message handler
// created by [WSOn]. It provides the message name and payload type for
// AsyncAPI schema generation, and a handle method for runtime dispatch.
type wsOnHandler interface {
	messageName() string
	messagePayloadType() reflect.Type
	handle(ctx context.Context, sender *WSSender, input any, data json.RawMessage) error
}

// onHandlerImpl is the concrete implementation of [wsOnHandler] created
// by the [WSOn] function.
type onHandlerImpl[In, Msg any] struct {
	name string
	fn   func(ctx context.Context, ws *WSSender, in In, msg Msg) error
}

func (h *onHandlerImpl[In, Msg]) messageName() string              { return h.name }
func (h *onHandlerImpl[In, Msg]) messagePayloadType() reflect.Type { return reflect.TypeFor[Msg]() }

func (h *onHandlerImpl[In, Msg]) handle(ctx context.Context, sender *WSSender, input any, data json.RawMessage) error {
	var msg Msg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("shiftapi: decode %q message: %w", h.name, err)
	}
	return h.fn(ctx, sender, input.(In), msg)
}

func (h *onHandlerImpl[In, Msg]) applyToWebsocket(cfg *websocketConfig) {
	cfg.handlers = append(cfg.handlers, h)
}

// On creates a [WebsocketHandler] that handles messages with the given type
// name. The handler function receives the parsed input, a [WSSender] for
// sending responses, and the decoded message payload.
//
// Each On handler provides both the runtime dispatch logic and the type
// information needed for AsyncAPI schema generation. The In type parameter
// must match the [Websocket] type parameter.
//
//	shiftapi.WSOn("message", func(ctx context.Context, ws *shiftapi.WSSender, in ChatInput, m UserMessage) error {
//	    return ws.Send(ctx, ChatMessage{User: "echo", Text: m.Text})
//	})
func WSOn[In, Msg any](name string, fn func(ctx context.Context, ws *WSSender, in In, msg Msg) error) WebsocketHandler {
	if name == "" {
		panic("shiftapi: WSOn name must not be empty")
	}
	return &onHandlerImpl[In, Msg]{name: name, fn: fn}
}

// sendsHandler registers send message types on the websocket config.
type sendsHandler struct {
	variants []MessageVariant
}

func (s *sendsHandler) applyToWebsocket(cfg *websocketConfig) {
	cfg.sendVariants = append(cfg.sendVariants, s.variants...)
}

// Sends registers named server-to-client message types for a WebSocket
// endpoint. [WSSender.Send] automatically wraps messages in a discriminated
// {"type", "data"} envelope based on the concrete Go type. WSSends is required.
//
// Each [MessageVariant] maps a type name to a payload type, producing a oneOf
// schema with a discriminator on the "type" field in the AsyncAPI spec.
//
//	shiftapi.WSSends(
//	    shiftapi.MessageType[ChatMessage]("chat"),
//	    shiftapi.MessageType[SystemMessage]("system"),
//	)
func WSSends(variants ...MessageVariant) WebsocketHandler {
	return &sendsHandler{variants: variants}
}

// WSMessages holds the assembled WebSocket endpoint configuration built by
// [Websocket]. It is passed as a positional argument to [HandleWS].
type WSMessages[In any] struct {
	cfg websocketConfig
}

// Websocket collects [WebsocketHandler] values ([WSOn] handlers and [WSSends]
// registrations) into a [WSMessages] configuration for [HandleWS].
//
// The type parameter In specifies the input struct for path, query, and header
// parameter parsing. Use struct{} when no input is needed.
//
//	shiftapi.Websocket[ChatInput](
//	    shiftapi.WSOn("message", func(ctx context.Context, ws *shiftapi.WSSender, in ChatInput, m UserMessage) error {
//	        return ws.Send(ctx, ChatMessage{Room: in.Room, Text: m.Text})
//	    }),
//	    shiftapi.WSSends(
//	        shiftapi.MessageType[ChatMessage]("chat"),
//	    ),
//	)
func Websocket[In any](handlers ...WebsocketHandler) WSMessages[In] {
	var cfg websocketConfig
	for _, h := range handlers {
		h.applyToWebsocket(&cfg)
	}
	if len(cfg.handlers) == 0 {
		panic("shiftapi: Websocket requires at least one WSOn handler")
	}
	if len(cfg.sendVariants) == 0 {
		panic("shiftapi: Websocket requires WSSends to define server-to-client message types")
	}
	return WSMessages[In]{cfg: cfg}
}

// MessageVariant describes a named WebSocket message type for AsyncAPI schema
// generation. Created by [MessageType] and passed to [WSSends].
type MessageVariant interface {
	messageName() string
	messagePayloadType() reflect.Type
}

type messageVariant[T any] struct {
	name string
}

func (m messageVariant[T]) messageName() string              { return m.name }
func (m messageVariant[T]) messagePayloadType() reflect.Type { return reflect.TypeFor[T]() }

// MessageType creates a [MessageVariant] that maps a message type name to a
// payload type T. Use with [WSSends] to register discriminated server-to-client
// message types for a WebSocket endpoint.
//
//	shiftapi.WSSends(
//	    shiftapi.MessageType[ChatMessage]("chat"),
//	    shiftapi.MessageType[SystemMessage]("system"),
//	)
func MessageType[T any](name string) MessageVariant {
	if name == "" {
		panic("shiftapi: MessageType name must not be empty")
	}
	return messageVariant[T]{name: name}
}

// runWSDispatchLoop runs the framework-managed receive loop for multi-type
// WebSocket endpoints. It reads discriminated messages, dispatches to the
// matching [WSOn] handler, and stops on close or error.
func runWSDispatchLoop[In any](ctx context.Context, conn *websocket.Conn, ws *WSSender, in In, dispatch map[string]wsOnHandler) {
	for {
		var envelope wsEvent
		if err := wsjson.Read(ctx, conn, &envelope); err != nil {
			if websocket.CloseStatus(err) != -1 {
				return // clean close
			}
			log.Printf("shiftapi: WS read error: %v", err)
			_ = conn.Close(websocket.StatusInternalError, "internal error")
			return
		}

		handler, ok := dispatch[envelope.Type]
		if !ok {
			log.Printf("shiftapi: unknown WS message type: %q", envelope.Type)
			continue
		}

		if err := handler.handle(ctx, ws, in, envelope.Data); err != nil {
			if websocket.CloseStatus(err) != -1 {
				return // handler triggered a close
			}
			log.Printf("shiftapi: WS handler error: %v", err)
			_ = conn.Close(websocket.StatusInternalError, "internal error")
			return
		}
	}
}

// WSOption configures a [HandleWS] route. General options like
// [WithRouteInfo], [WithError], and [WithMiddleware] implement both
// [RouteOption] and [WSOption]. WebSocket-specific options like
// [WithWSAcceptOptions] implement only [WSOption].
type WSOption interface {
	applyToWS(*wsRouteConfig)
}

// wsRouteConfig holds the registration-time configuration for a WebSocket
// route, built from [WSOption] values.
type wsRouteConfig struct {
	info              *RouteInfo
	errors            []errorEntry
	middleware        []func(http.Handler) http.Handler
	staticRespHeaders []staticResponseHeader
	wsAcceptOptions   *WSAcceptOptions
}

func (c *wsRouteConfig) addError(e errorEntry) {
	c.errors = append(c.errors, e)
}

func (c *wsRouteConfig) addMiddleware(mw []func(http.Handler) http.Handler) {
	c.middleware = append(c.middleware, mw...)
}

func (c *wsRouteConfig) addStaticResponseHeader(h staticResponseHeader) {
	c.staticRespHeaders = append(c.staticRespHeaders, h)
}

func applyWSOptions(opts []WSOption) wsRouteConfig {
	var cfg wsRouteConfig
	for _, opt := range opts {
		opt.applyToWS(&cfg)
	}
	return cfg
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
//	shiftapi.HandleWS(api, "GET /ws",
//	    shiftapi.Websocket[struct{}](...),
//	    shiftapi.WithWSAcceptOptions(shiftapi.WSAcceptOptions{
//	        Subprotocols:   []string{"graphql-ws"},
//	        OriginPatterns: []string{"example.com"},
//	    }),
//	)
func WithWSAcceptOptions(opts WSAcceptOptions) wsOptionFunc {
	return func(cfg *wsRouteConfig) {
		cfg.wsAcceptOptions = &opts
	}
}

// wsOptionFunc is a function that implements [WSOption].
type wsOptionFunc func(*wsRouteConfig)

func (f wsOptionFunc) applyToWS(cfg *wsRouteConfig) { f(cfg) }

// Ensure that the shared Option type also implements WSOption.
func (f Option) applyToWS(cfg *wsRouteConfig) { f(cfg) }
