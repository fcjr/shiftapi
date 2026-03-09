package shiftapi

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// WSHandlerFunc is a handler function for WebSocket endpoints. It receives
// the parsed input and a typed [WSConn] for bidirectional communication.
//
// The handler is called after the WebSocket upgrade has completed. Use
// [WSConn.Send] and [WSConn.Receive] for typed message exchange and return
// nil when the connection is complete. If the handler returns an error, the
// connection is closed with [WSStatusInternalError] and the error is logged.
//
// Input parsing and validation errors (before upgrade) produce a normal
// JSON error response, identical to [Handle].
type WSHandlerFunc[In, Send, Recv any] func(r *http.Request, in In, ws *WSConn[Send, Recv]) error

// WSConn is a typed WebSocket connection wrapper. It provides type-safe
// Send and Receive methods over the underlying [websocket.Conn].
//
// For single-type endpoints, use [WSConn.Send] and [WSConn.Receive] for
// direct typed messaging. For multi-type endpoints (registered with
// [WithSendMessages] / [WithRecvMessages]), use [WSConn.SendEvent] and
// [WSConn.ReceiveEvent] which wrap messages in a discriminated
// {"type", "data"} envelope.
//
// WSConn is created internally by [HandleWS] and should not be constructed
// directly.
type WSConn[Send, Recv any] struct {
	conn *websocket.Conn
}

// Send writes a JSON-encoded message to the WebSocket connection.
// Use this for single-type endpoints without [WithSendMessages].
func (ws *WSConn[Send, Recv]) Send(ctx context.Context, v Send) error {
	return wsjson.Write(ctx, ws.conn, v)
}

// Receive reads and JSON-decodes a message from the WebSocket connection.
// Use this for single-type endpoints without [WithRecvMessages].
func (ws *WSConn[Send, Recv]) Receive(ctx context.Context) (Recv, error) {
	var v Recv
	err := wsjson.Read(ctx, ws.conn, &v)
	return v, err
}

// SendEvent writes a discriminated JSON message to the WebSocket connection.
// The value is wrapped in an envelope: {"type": eventType, "data": v}.
// Use this for multi-type endpoints registered with [WithSendMessages].
func (ws *WSConn[Send, Recv]) SendEvent(ctx context.Context, eventType string, v Send) error {
	envelope := wsEnvelope[Send]{Type: eventType, Data: v}
	return wsjson.Write(ctx, ws.conn, envelope)
}

// ReceiveEvent reads a discriminated JSON message from the WebSocket
// connection. It expects an envelope: {"type": "...", "data": ...} and
// returns a [WSEvent] with the type name and raw data. Call
// [WSEvent.Decode] to unmarshal into the concrete type.
// Use this for multi-type endpoints registered with [WithRecvMessages].
func (ws *WSConn[Send, Recv]) ReceiveEvent(ctx context.Context) (WSEvent, error) {
	var msg WSEvent
	err := wsjson.Read(ctx, ws.conn, &msg)
	return msg, err
}

// Close closes the WebSocket connection with the given status code and reason.
func (ws *WSConn[Send, Recv]) Close(status WSStatusCode, reason string) error {
	return ws.conn.Close(websocket.StatusCode(status), reason)
}

// Conn returns the underlying [websocket.Conn] for advanced use cases
// such as binary frames, ping/pong, or custom close handling. Most
// handlers should use [Send], [Receive], [SendEvent], and [ReceiveEvent]
// instead.
func (ws *WSConn[Send, Recv]) Conn() *websocket.Conn {
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

// WSEvent represents a received discriminated WebSocket message.
// Use [WSEvent.Decode] to unmarshal the data into a concrete type
// after switching on [WSEvent.Type].
type WSEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Decode unmarshals the message data into the given value.
func (m WSEvent) Decode(v any) error {
	return json.Unmarshal(m.Data, v)
}

// MessageVariant describes a named WebSocket message type for AsyncAPI schema
// generation. Created by [MessageType] and passed to [WithSendMessages] or
// [WithRecvMessages].
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
// payload type T. Use with [WithSendMessages] or [WithRecvMessages] to
// register discriminated message types for a WebSocket endpoint.
//
//	shiftapi.HandleWS(api, "GET /chat", chatHandler,
//	    shiftapi.WithSendMessages(
//	        shiftapi.MessageType[ChatMessage]("chat"),
//	        shiftapi.MessageType[SystemMessage]("system"),
//	    ),
//	    shiftapi.WithRecvMessages(
//	        shiftapi.MessageType[UserMessage]("message"),
//	        shiftapi.MessageType[UserCommand]("command"),
//	    ),
//	)
func MessageType[T any](name string) MessageVariant {
	if name == "" {
		panic("shiftapi: MessageType name must not be empty")
	}
	return messageVariant[T]{name: name}
}
