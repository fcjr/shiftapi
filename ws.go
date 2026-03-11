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
	ctx          context.Context
	sendVariants map[reflect.Type]string // nil = raw mode
}

// Send writes a JSON-encoded message to the WebSocket connection. The value
// is automatically wrapped in a {"type": name, "data": value} envelope based
// on its concrete Go type, using the types registered via [WSSends].
func (ws *WSSender) Send(v any) error {
	name, ok := ws.sendVariants[reflect.TypeOf(v)]
	if !ok {
		return fmt.Errorf("shiftapi: unregistered send type %T; register with WSSends", v)
	}
	envelope := wsEnvelope[any]{Type: name, Data: v}
	return wsjson.Write(ws.ctx, ws.conn, envelope)
}

// Close closes the WebSocket connection with the given status code and reason.
func (ws *WSSender) Close(status WSStatusCode, reason string) error {
	return ws.conn.Close(websocket.StatusCode(status), reason)
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

// websocketConfig holds the internal configuration for a WebSocket endpoint.
type websocketConfig struct {
	handlers     []wsOnHandler
	sendVariants []WSMessageVariant
	setup        func(r *http.Request, ws *WSSender, input any) (any, error)
}

// wsOnHandler is the internal interface for a typed message handler
// created by [WSOn]. It provides the message name and payload type for
// AsyncAPI schema generation, and a handle method for runtime dispatch.
type wsOnHandler interface {
	messageName() string
	messagePayloadType() reflect.Type
	handle(r *http.Request, sender *WSSender, state any, data json.RawMessage) error
}

// onHandlerImpl is the concrete implementation of [wsOnHandler] created
// by the [WSOn] function.
type onHandlerImpl[State, Msg any] struct {
	name string
	fn   func(r *http.Request, ws *WSSender, state State, msg Msg) error
}

func (h *onHandlerImpl[State, Msg]) messageName() string              { return h.name }
func (h *onHandlerImpl[State, Msg]) messagePayloadType() reflect.Type { return reflect.TypeFor[Msg]() }

func (h *onHandlerImpl[State, Msg]) handle(r *http.Request, sender *WSSender, state any, data json.RawMessage) error {
	var msg Msg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("shiftapi: decode %q message: %w", h.name, err)
	}
	return h.fn(r, sender, state.(State), msg)
}

// WSMessages holds the WebSocket endpoint configuration. Create one with
// [Websocket], passing a setup function, [WSSends], and [WSOn] handlers.
// Pass it to [HandleWS] to register the route.
type WSMessages[In any] struct {
	cfg *websocketConfig
}

// WSHandler is a typed message handler for a [Websocket] endpoint. Create
// one with [WSOn]. The State type parameter must match the setup function's
// return type.
type WSHandler[State any] struct {
	name string
	impl wsOnHandler
}

// WSSends declares the named server-to-client message types for a WebSocket
// endpoint. Pass [WSMessageType] values to register each type.
// [WSSender.Send] automatically wraps messages in a discriminated
// {"type", "data"} envelope based on the concrete Go type.
//
//	shiftapi.WSSends(
//	    shiftapi.WSMessageType[ChatMessage]("chat"),
//	    shiftapi.WSMessageType[SystemMessage]("system"),
//	)
type WSSends []WSMessageVariant

// Websocket creates a new WebSocket endpoint configuration. The type
// parameters In and State are both inferred from the setup function:
// In from the input parameter, State from the return value. Handlers
// receive the State value returned by setup on each connection.
//
// Use struct{} for both In and State when no input or state is needed.
//
//	shiftapi.HandleWS(api, "GET /echo",
//	    shiftapi.Websocket(
//	        func(r *http.Request, s *shiftapi.WSSender, _ struct{}) (struct{}, error) {
//	            return struct{}{}, nil
//	        },
//	        shiftapi.WSSends(shiftapi.WSMessageType[ServerMsg]("server")),
//	        shiftapi.WSOn("echo", func(r *http.Request, s *shiftapi.WSSender, _ struct{}, msg ClientMsg) error {
//	            return s.Send(ServerMsg{Text: msg.Text})
//	        }),
//	    ),
//	)
func Websocket[In, State any](setup func(r *http.Request, sender *WSSender, in In) (State, error), sends WSSends, handlers ...WSHandler[State]) *WSMessages[In] {
	cfg := &websocketConfig{
		sendVariants: []WSMessageVariant(sends),
		setup: func(r *http.Request, ws *WSSender, input any) (any, error) {
			return setup(r, ws, input.(In))
		},
	}
	for _, h := range handlers {
		cfg.handlers = append(cfg.handlers, h.impl)
	}
	return &WSMessages[In]{cfg: cfg}
}

// WSOn creates a typed message handler for a [Websocket] endpoint.
// The State and Msg type parameters are inferred from the handler function.
// State must match the setup function's return type.
//
//	shiftapi.WSOn("message", func(r *http.Request, s *shiftapi.WSSender, state *Room, msg UserMessage) error {
//	    state.Broadcast(msg)
//	    return nil
//	})
func WSOn[State, Msg any](name string, fn func(r *http.Request, sender *WSSender, state State, msg Msg) error) WSHandler[State] {
	if name == "" {
		panic("shiftapi: WSOn name must not be empty")
	}
	return WSHandler[State]{
		name: name,
		impl: &onHandlerImpl[State, Msg]{name: name, fn: fn},
	}
}

// WSMessageVariant describes a named WebSocket message type for AsyncAPI schema
// generation. Created by [WSMessageType] and passed to [WSSends].
type WSMessageVariant interface {
	messageName() string
	messagePayloadType() reflect.Type
}

type messageVariant[T any] struct {
	name string
}

func (m messageVariant[T]) messageName() string              { return m.name }
func (m messageVariant[T]) messagePayloadType() reflect.Type { return reflect.TypeFor[T]() }

// WSMessageType creates a [WSMessageVariant] that maps a message type name to a
// payload type T. Use with [WSSends] to register discriminated server-to-client
// message types for a WebSocket endpoint.
//
//	shiftapi.WSSends(
//	    shiftapi.WSMessageType[ChatMessage]("chat"),
//	    shiftapi.WSMessageType[SystemMessage]("system"),
//	)
func WSMessageType[T any](name string) WSMessageVariant {
	if name == "" {
		panic("shiftapi: WSMessageType name must not be empty")
	}
	return messageVariant[T]{name: name}
}

// wsCallbacks holds the optional user callbacks for the dispatch loop.
type wsCallbacks struct {
	onError      func(r *http.Request, ws *WSSender, err error)
	onUnknownMsg func(r *http.Request, ws *WSSender, msgType string, data json.RawMessage)
}

// runWSDispatchLoop runs the framework-managed receive loop for multi-type
// WebSocket endpoints. It reads discriminated messages, dispatches to the
// matching [WSOn] handler, and stops on close or error.
func runWSDispatchLoop(r *http.Request, conn *websocket.Conn, ws *WSSender, state any, dispatch map[string]wsOnHandler, cb wsCallbacks) {
	ctx := r.Context()
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
			if cb.onUnknownMsg != nil {
				cb.onUnknownMsg(r, ws, envelope.Type, envelope.Data)
			} else {
				log.Printf("shiftapi: unknown WS message type: %q", envelope.Type)
			}
			continue
		}

		if err := handler.handle(r, ws, state, envelope.Data); err != nil {
			if websocket.CloseStatus(err) != -1 {
				return // handler triggered a close
			}
			if cb.onError != nil {
				cb.onError(r, ws, err)
			} else {
				log.Printf("shiftapi: WS handler error: %v", err)
				_ = conn.Close(websocket.StatusInternalError, "internal error")
			}
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
	onError           func(r *http.Request, ws *WSSender, err error)
	onUnknownMsg      func(r *http.Request, ws *WSSender, msgType string, data json.RawMessage)
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
//	shiftapi.HandleWS(api, "GET /ws", ws,
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

// WithWSOnError registers an error callback for [HandleWS] routes. The
// callback is invoked when a [WSOn] handler returns a non-close error.
// If not set, the framework logs the error and closes the connection with
// [WSStatusInternalError].
//
// When set, the callback owns the close decision — the framework will not
// auto-close after the callback returns.
//
//	shiftapi.HandleWS(api, "GET /ws", ws,
//	    shiftapi.WithWSOnError(func(r *http.Request, s *shiftapi.WSSender, err error) {
//	        s.Send(ErrorMsg{Message: err.Error()})
//	        s.Close(shiftapi.WSStatusInternalError, "handler error")
//	    }),
//	)
func WithWSOnError(fn func(r *http.Request, sender *WSSender, err error)) wsOptionFunc {
	return func(cfg *wsRouteConfig) {
		cfg.onError = fn
	}
}

// WithWSOnUnknownMessage registers a callback for unrecognized message types
// on [HandleWS] routes. The callback is invoked when the client sends a message
// whose "type" field does not match any registered [WSOn] handler. If not set,
// the framework logs the unknown type and continues reading.
//
//	shiftapi.HandleWS(api, "GET /ws", ws,
//	    shiftapi.WithWSOnUnknownMessage(func(r *http.Request, s *shiftapi.WSSender, msgType string, data json.RawMessage) {
//	        log.Printf("unknown message type: %s", msgType)
//	    }),
//	)
func WithWSOnUnknownMessage(fn func(r *http.Request, sender *WSSender, msgType string, data json.RawMessage)) wsOptionFunc {
	return func(cfg *wsRouteConfig) {
		cfg.onUnknownMsg = fn
	}
}

// wsOptionFunc is a function that implements [WSOption].
type wsOptionFunc func(*wsRouteConfig)

func (f wsOptionFunc) applyToWS(cfg *wsRouteConfig) { f(cfg) }

// Ensure that the shared Option type also implements WSOption.
func (f Option) applyToWS(cfg *wsRouteConfig) { f(cfg) }
