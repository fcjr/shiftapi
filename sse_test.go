package shiftapi_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fcjr/shiftapi"
)

type sseMessage struct {
	Text string `json:"text"`
}

func TestSSEWriter_Send(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleSSE(api, "GET /events", func(r *http.Request, _ struct{}, sse *shiftapi.SSEWriter[sseMessage]) error {
		if err := sse.Send(sseMessage{Text: "hello"}); err != nil {
			return err
		}
		return sse.Send(sseMessage{Text: "world"})
	}, shiftapi.SSESends(
		shiftapi.SSEEventType[sseMessage]("message"),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/events", nil)
	api.ServeHTTP(w, r)

	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %q, want %q", w.Header().Get("Content-Type"), "text/event-stream")
	}
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("Cache-Control = %q, want %q", w.Header().Get("Cache-Control"), "no-cache")
	}
	if w.Header().Get("Connection") != "keep-alive" {
		t.Errorf("Connection = %q, want %q", w.Header().Get("Connection"), "keep-alive")
	}

	events := parseSSEEvents(t, w.Body.String())
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].Event != "message" {
		t.Errorf("event[0].Event = %q, want %q", events[0].Event, "message")
	}
	if events[0].Data != `{"text":"hello"}` {
		t.Errorf("event[0].Data = %q, want %q", events[0].Data, `{"text":"hello"}`)
	}
	if events[1].Event != "message" {
		t.Errorf("event[1].Event = %q, want %q", events[1].Event, "message")
	}
	if events[1].Data != `{"text":"world"}` {
		t.Errorf("event[1].Data = %q, want %q", events[1].Data, `{"text":"world"}`)
	}
}

func TestHandleSSE_InputParsing(t *testing.T) {
	api := shiftapi.New()

	type Input struct {
		Channel string `query:"channel" validate:"required"`
	}

	shiftapi.HandleSSE(api, "GET /events", func(r *http.Request, in Input, sse *shiftapi.SSEWriter[sseMessage]) error {
		return sse.Send(sseMessage{Text: "channel=" + in.Channel})
	}, shiftapi.SSESends(
		shiftapi.SSEEventType[sseMessage]("message"),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/events?channel=general", nil)
	api.ServeHTTP(w, r)

	events := parseSSEEvents(t, w.Body.String())
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Data != `{"text":"channel=general"}` {
		t.Errorf("event[0].Data = %q, want %q", events[0].Data, `{"text":"channel=general"}`)
	}
}

func TestHandleSSE_InputValidationError(t *testing.T) {
	api := shiftapi.New()

	type Input struct {
		Channel string `query:"channel" validate:"required"`
	}

	shiftapi.HandleSSE(api, "GET /events", func(r *http.Request, in Input, sse *shiftapi.SSEWriter[sseMessage]) error {
		return sse.Send(sseMessage{Text: "should not reach"})
	}, shiftapi.SSESends(
		shiftapi.SSEEventType[sseMessage]("message"),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/events", nil) // missing required channel
	api.ServeHTTP(w, r)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleSSE_ErrorBeforeWrite(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleSSE(api, "GET /events", func(r *http.Request, _ struct{}, sse *shiftapi.SSEWriter[sseMessage]) error {
		return fmt.Errorf("something went wrong")
	}, shiftapi.SSESends(
		shiftapi.SSEEventType[sseMessage]("message"),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/events", nil)
	api.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleSSE_PathParams(t *testing.T) {
	api := shiftapi.New()

	type Input struct {
		ID string `path:"id"`
	}

	shiftapi.HandleSSE(api, "GET /streams/{id}", func(r *http.Request, in Input, sse *shiftapi.SSEWriter[sseMessage]) error {
		return sse.Send(sseMessage{Text: "id=" + in.ID})
	}, shiftapi.SSESends(
		shiftapi.SSEEventType[sseMessage]("message"),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/streams/abc", nil)
	api.ServeHTTP(w, r)

	events := parseSSEEvents(t, w.Body.String())
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Data != `{"text":"id=abc"}` {
		t.Errorf("event[0].Data = %q, want %q", events[0].Data, `{"text":"id=abc"}`)
	}
}

func TestHandleSSE_OpenAPISpec(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleSSE(api, "GET /events", func(r *http.Request, _ struct{}, sse *shiftapi.SSEWriter[sseMessage]) error {
		return nil
	}, shiftapi.SSESends(
		shiftapi.SSEEventType[sseMessage]("message"),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/openapi.json", nil)
	api.ServeHTTP(w, r)

	var spec map[string]any
	if err := json.NewDecoder(w.Body).Decode(&spec); err != nil {
		t.Fatalf("decode spec: %v", err)
	}

	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatal("no paths in spec")
	}
	eventsPath, ok := paths["/events"].(map[string]any)
	if !ok {
		t.Fatal("no /events path in spec")
	}
	getOp, ok := eventsPath["get"].(map[string]any)
	if !ok {
		t.Fatal("no GET operation on /events")
	}
	responses, ok := getOp["responses"].(map[string]any)
	if !ok {
		t.Fatal("no responses in GET /events")
	}
	resp200, ok := responses["200"].(map[string]any)
	if !ok {
		t.Fatal("no 200 response in GET /events")
	}
	content, ok := resp200["content"].(map[string]any)
	if !ok {
		t.Fatal("no content in 200 response")
	}
	if _, ok := content["text/event-stream"]; !ok {
		t.Error("200 response missing text/event-stream content type")
	}
}

// --- Multi-event (SSESends) tests ---

type chatEvent interface{ chatEvent() }

type messageData struct {
	User string `json:"user"`
	Text string `json:"text"`
}

func (messageData) chatEvent() {}

type joinData struct {
	User string `json:"user"`
}

func (joinData) chatEvent() {}

func TestHandleSSE_SSESends_AutoWrapSend(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleSSE(api, "GET /chat", func(r *http.Request, _ struct{}, sse *shiftapi.SSEWriter[chatEvent]) error {
		if err := sse.Send(messageData{User: "alice", Text: "hi"}); err != nil {
			return err
		}
		return sse.Send(joinData{User: "bob"})
	}, shiftapi.SSESends(
		shiftapi.SSEEventType[messageData]("message"),
		shiftapi.SSEEventType[joinData]("join"),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/chat", nil)
	api.ServeHTTP(w, r)

	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %q, want %q", w.Header().Get("Content-Type"), "text/event-stream")
	}

	events := parseSSEEvents(t, w.Body.String())
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].Event != "message" {
		t.Errorf("event[0].Event = %q, want %q", events[0].Event, "message")
	}
	if events[0].Data != `{"user":"alice","text":"hi"}` {
		t.Errorf("event[0].Data = %q, want %q", events[0].Data, `{"user":"alice","text":"hi"}`)
	}
	if events[1].Event != "join" {
		t.Errorf("event[1].Event = %q, want %q", events[1].Event, "join")
	}
	if events[1].Data != `{"user":"bob"}` {
		t.Errorf("event[1].Data = %q, want %q", events[1].Data, `{"user":"bob"}`)
	}
}

func TestHandleSSE_SSESends_OpenAPISpec(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleSSE(api, "GET /chat", func(r *http.Request, _ struct{}, sse *shiftapi.SSEWriter[chatEvent]) error {
		return nil
	}, shiftapi.SSESends(
		shiftapi.SSEEventType[messageData]("message"),
		shiftapi.SSEEventType[joinData]("join"),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/openapi.json", nil)
	api.ServeHTTP(w, r)

	var spec map[string]any
	if err := json.NewDecoder(w.Body).Decode(&spec); err != nil {
		t.Fatalf("decode spec: %v", err)
	}

	// Navigate to the text/event-stream schema
	paths := spec["paths"].(map[string]any)
	chatPath := paths["/chat"].(map[string]any)
	getOp := chatPath["get"].(map[string]any)
	responses := getOp["responses"].(map[string]any)
	resp200 := responses["200"].(map[string]any)
	content := resp200["content"].(map[string]any)
	sse := content["text/event-stream"].(map[string]any)
	schema := sse["schema"].(map[string]any)

	// Should have oneOf
	oneOf, ok := schema["oneOf"].([]any)
	if !ok {
		t.Fatal("schema missing oneOf")
	}
	if len(oneOf) != 2 {
		t.Fatalf("oneOf has %d items, want 2", len(oneOf))
	}

	// Should have discriminator
	disc, ok := schema["discriminator"].(map[string]any)
	if !ok {
		t.Fatal("schema missing discriminator")
	}
	if disc["propertyName"] != "event" {
		t.Errorf("discriminator.propertyName = %q, want %q", disc["propertyName"], "event")
	}

	// Check first variant (message)
	v0 := oneOf[0].(map[string]any)
	v0Props := v0["properties"].(map[string]any)
	v0Event := v0Props["event"].(map[string]any)
	v0Enum := v0Event["enum"].([]any)
	if len(v0Enum) != 1 || v0Enum[0] != "message" {
		t.Errorf("variant 0 event enum = %v, want [message]", v0Enum)
	}
	v0Required := v0["required"].([]any)
	if len(v0Required) != 2 {
		t.Errorf("variant 0 required = %v, want [event data]", v0Required)
	}

	// Check second variant (join)
	v1 := oneOf[1].(map[string]any)
	v1Props := v1["properties"].(map[string]any)
	v1Event := v1Props["event"].(map[string]any)
	v1Enum := v1Event["enum"].([]any)
	if len(v1Enum) != 1 || v1Enum[0] != "join" {
		t.Errorf("variant 1 event enum = %v, want [join]", v1Enum)
	}
}

func TestHandleSSE_MissingSSESendsPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for missing SSESends")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "requires SSESends") {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()
	api := shiftapi.New()
	shiftapi.HandleSSE(api, "GET /events", func(r *http.Request, _ struct{}, sse *shiftapi.SSEWriter[sseMessage]) error {
		return nil
	})
}

func TestEventType_EmptyNamePanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for empty event name")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "must not be empty") {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()
	shiftapi.SSEEventType[sseMessage]("")
}

func TestSSESends_DuplicateNamePanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for duplicate event name")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "duplicate event name") {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()
	api := shiftapi.New()
	shiftapi.HandleSSE(api, "GET /dup", func(r *http.Request, _ struct{}, sse *shiftapi.SSEWriter[chatEvent]) error {
		return nil
	}, shiftapi.SSESends(
		shiftapi.SSEEventType[messageData]("same"),
		shiftapi.SSEEventType[joinData]("same"),
	))
}

func TestHandleSSE_SSESends_UnregisteredTypeErrors(t *testing.T) {
	api := shiftapi.New()

	// joinData satisfies chatEvent but is not registered in SSESends.
	shiftapi.HandleSSE(api, "GET /chat", func(r *http.Request, _ struct{}, sse *shiftapi.SSEWriter[chatEvent]) error {
		return sse.Send(joinData{User: "bob"})
	}, shiftapi.SSESends(
		shiftapi.SSEEventType[messageData]("message"),
	))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/chat", nil)
	api.ServeHTTP(w, r)

	// The handler returned an error before any events were written,
	// so we expect a 500 error response.
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// sseEvent represents a parsed SSE event.
type sseEvent struct {
	Event string
	Data  string
}

// parseSSEEvents parses SSE-formatted text into events.
func parseSSEEvents(t *testing.T, body string) []sseEvent {
	t.Helper()
	var events []sseEvent
	scanner := bufio.NewScanner(strings.NewReader(body))
	var current sseEvent
	var dataLines []string
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			current.Event = line[7:]
		case strings.HasPrefix(line, "data: "):
			dataLines = append(dataLines, line[6:])
		case line == "":
			if len(dataLines) > 0 {
				current.Data = strings.Join(dataLines, "\n")
				events = append(events, current)
				current = sseEvent{}
				dataLines = nil
			}
		}
	}
	return events
}
