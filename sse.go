package shiftapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
)

// SSEHandlerFunc is a handler function for Server-Sent Events. It receives
// the parsed input and a typed [SSEWriter] for sending events to the client.
//
// The handler should send events via [SSEWriter.Send] or [SSEWriter.SendEvent]
// and return nil when the stream is complete. If the handler returns an error
// before any events have been sent, a JSON error response is written. If the
// error occurs after events have been sent the error is logged (the response
// has already started).
type SSEHandlerFunc[In, Event any] func(r *http.Request, in In, sse *SSEWriter[Event]) error

// SSEWriter writes typed Server-Sent Events to the client. It is created
// internally by [HandleSSE] and should not be constructed directly.
//
// On the first call to [Send] or [SendEvent], SSEWriter sets the required SSE
// headers (Content-Type, Cache-Control, Connection) before writing data.
type SSEWriter[Event any] struct {
	w            http.ResponseWriter
	rc           *http.ResponseController
	started      bool
	sendVariants map[reflect.Type]string // nil = no auto-wrap
}

// Send writes an SSE event. When [WithEvents] is used, the event name is
// automatically determined from the concrete Go type of v — the value is
// written as:
//
//	event: {name}\ndata: {json}\n\n
//
// Without [WithEvents], the value is written as a data-only event:
//
//	data: {json}\n\n
//
// The response is flushed after each event.
func (s *SSEWriter[Event]) Send(v Event) error {
	if s.sendVariants != nil {
		name, ok := s.sendVariants[reflect.TypeOf(v)]
		if !ok {
			return fmt.Errorf("shiftapi: unregistered SSE event type %T; register with WithEvents", v)
		}
		return s.sendEvent(name, v)
	}
	return s.sendData(v)
}

// SendEvent writes a named SSE event. The value is JSON-encoded and written as:
//
//	event: {name}\ndata: {json}\n\n
//
// The response is flushed after each event.
func (s *SSEWriter[Event]) SendEvent(event string, v Event) error {
	return s.sendEvent(event, v)
}

func (s *SSEWriter[Event]) sendData(v Event) error {
	s.writeHeaders()
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("shiftapi: SSE marshal error: %w", err)
	}
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return fmt.Errorf("shiftapi: SSE write error: %w", err)
	}
	return s.rc.Flush()
}

func (s *SSEWriter[Event]) sendEvent(event string, v Event) error {
	s.writeHeaders()
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("shiftapi: SSE marshal error: %w", err)
	}
	if _, err := fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return fmt.Errorf("shiftapi: SSE write error: %w", err)
	}
	return s.rc.Flush()
}

// writeHeaders sets SSE headers on the first write.
func (s *SSEWriter[Event]) writeHeaders() {
	if s.started {
		return
	}
	s.started = true
	h := s.w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
}

// EventVariant describes a named SSE event type for OpenAPI schema generation.
// Created by [EventType] and passed to [WithEvents].
type EventVariant interface {
	eventName() string
	eventPayloadType() reflect.Type
}

type eventVariant[T any] struct {
	name string
}

func (e eventVariant[T]) eventName() string              { return e.name }
func (e eventVariant[T]) eventPayloadType() reflect.Type { return reflect.TypeFor[T]() }

// EventType creates an [EventVariant] that maps an SSE event name to a payload
// type T. Use with [WithEvents] to register discriminated event types for a
// single SSE endpoint. The OpenAPI spec will contain a oneOf schema with a
// discriminator, and the generated TypeScript client will yield a discriminated
// union type.
//
//	shiftapi.HandleSSE(api, "GET /chat", chatHandler,
//	    shiftapi.WithEvents(
//	        shiftapi.EventType[MessageData]("message"),
//	        shiftapi.EventType[JoinData]("join"),
//	    ),
//	)
func EventType[T any](name string) EventVariant {
	if name == "" {
		panic("shiftapi: EventType name must not be empty")
	}
	return eventVariant[T]{name: name}
}
