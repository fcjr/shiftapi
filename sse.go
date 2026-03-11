package shiftapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
)

// SSEHandlerFunc is a handler function for Server-Sent Events. It receives
// the parsed input and an [SSEWriter] for sending events to the client.
//
// The handler should send events via [SSEWriter.Send] and return nil when
// the stream is complete. If the handler returns an error before any events
// have been sent, a JSON error response is written. If the error occurs
// after events have been sent the error is logged (the response has already
// started).
type SSEHandlerFunc[In any] func(r *http.Request, in In, sse *SSEWriter) error

// SSEWriter writes Server-Sent Events to the client. It is created
// internally by [HandleSSE] and should not be constructed directly.
//
// [SSEWriter.Send] automatically determines the event name from the concrete
// Go type registered via [SSESends]. On the first call, SSEWriter sets the
// required SSE headers (Content-Type, Cache-Control, Connection).
type SSEWriter struct {
	w            http.ResponseWriter
	rc           *http.ResponseController
	started      bool
	sendVariants map[reflect.Type]string
}

// Send writes an SSE event. The event name is automatically determined from the
// concrete Go type registered via [SSESends]:
//
//	event: {name}\ndata: {json}\n\n
//
// The response is flushed after each event.
func (s *SSEWriter) Send(v any) error {
	name, ok := s.sendVariants[reflect.TypeOf(v)]
	if !ok {
		return fmt.Errorf("shiftapi: unregistered SSE event type %T; register with SSESends", v)
	}
	s.writeHeaders()
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("shiftapi: SSE marshal error: %w", err)
	}
	if _, err := fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", name, data); err != nil {
		return fmt.Errorf("shiftapi: SSE write error: %w", err)
	}
	return s.rc.Flush()
}

// writeHeaders sets SSE headers on the first write.
func (s *SSEWriter) writeHeaders() {
	if s.started {
		return
	}
	s.started = true
	h := s.w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
}

// SSEEventVariant describes a named SSE event type for OpenAPI schema generation.
// Created by [SSEEventType] and passed to [SSESends].
type SSEEventVariant interface {
	eventName() string
	eventPayloadType() reflect.Type
}

type sseEventVariant[T any] struct {
	name string
}

func (e sseEventVariant[T]) eventName() string              { return e.name }
func (e sseEventVariant[T]) eventPayloadType() reflect.Type { return reflect.TypeFor[T]() }

// SSEEventType creates an [SSEEventVariant] that maps an SSE event name to a
// payload type T. Use with [SSESends] to register event types for a [HandleSSE]
// endpoint. The OpenAPI spec will contain a oneOf schema with a discriminator,
// and the generated TypeScript client will yield a discriminated union type.
//
//	shiftapi.HandleSSE(api, "GET /chat", chatHandler,
//	    shiftapi.SSESends(
//	        shiftapi.SSEEventType[MessageData]("message"),
//	        shiftapi.SSEEventType[JoinData]("join"),
//	    ),
//	)
func SSEEventType[T any](name string) SSEEventVariant {
	if name == "" {
		panic("shiftapi: SSEEventType name must not be empty")
	}
	return sseEventVariant[T]{name: name}
}

// SSEOption configures a [HandleSSE] route. General options like
// [WithRouteInfo], [WithError], and [WithMiddleware] implement both
// [RouteOption] and [SSEOption]. SSE-specific options like [SSESends]
// implement only [SSEOption].
type SSEOption interface {
	applyToSSE(*sseRouteConfig)
}

// sseRouteConfig holds the registration-time configuration for an SSE
// route, built from [SSEOption] values.
type sseRouteConfig struct {
	info              *RouteInfo
	errors            []errorEntry
	middleware        []func(http.Handler) http.Handler
	staticRespHeaders []staticResponseHeader
	eventVariants     []SSEEventVariant
}

func (c *sseRouteConfig) addError(e errorEntry) {
	c.errors = append(c.errors, e)
}

func (c *sseRouteConfig) addMiddleware(mw []func(http.Handler) http.Handler) {
	c.middleware = append(c.middleware, mw...)
}

func (c *sseRouteConfig) addStaticResponseHeader(h staticResponseHeader) {
	c.staticRespHeaders = append(c.staticRespHeaders, h)
}

func applySSEOptions(opts []SSEOption) sseRouteConfig {
	var cfg sseRouteConfig
	for _, opt := range opts {
		opt.applyToSSE(&cfg)
	}
	return cfg
}

// sseOptionFunc is a function that implements [SSEOption].
type sseOptionFunc func(*sseRouteConfig)

func (f sseOptionFunc) applyToSSE(cfg *sseRouteConfig) { f(cfg) }

// SSESends registers named SSE event types for auto-wrap and OpenAPI schema
// generation. Each [SSEEventVariant] maps an event name to a payload type,
// producing a oneOf schema with a discriminator in the OpenAPI spec. The
// generated TypeScript client yields a discriminated union type. SSESends
// is required for [HandleSSE].
//
// When SSESends is used, [SSEWriter.Send] automatically determines the event
// name from the concrete Go type.
//
//	shiftapi.HandleSSE(api, "GET /chat", chatHandler,
//	    shiftapi.SSESends(
//	        shiftapi.SSEEventType[MessageData]("message"),
//	        shiftapi.SSEEventType[JoinData]("join"),
//	    ),
//	)
func SSESends(variants ...SSEEventVariant) sseOptionFunc {
	return func(cfg *sseRouteConfig) {
		cfg.eventVariants = append(cfg.eventVariants, variants...)
	}
}

// Ensure that the shared Option type also implements SSEOption.
func (f Option) applyToSSE(cfg *sseRouteConfig) { f(cfg) }
