package shiftapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/fcjr/shiftapi"
)

type wsServerMsg struct {
	Text string `json:"text"`
}

type wsClientMsg struct {
	Text string `json:"text"`
}

func TestHandleWS_AsyncAPISpec(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket[struct{}](
			shiftapi.WSOn("echo", func(ctx context.Context, ws *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
				return ws.Send(ctx, wsServerMsg{Text: msg.Text})
			}),
			shiftapi.WSSends(
				shiftapi.MessageType[wsServerMsg]("server"),
			),
		),
		shiftapi.WithRouteInfo(shiftapi.RouteInfo{
			Summary: "Echo WS",
			Tags:    []string{"websocket"},
		}),
	)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/asyncapi.json", nil)
	api.ServeHTTP(w, r)

	var spec map[string]any
	if err := json.NewDecoder(w.Body).Decode(&spec); err != nil {
		t.Fatalf("decode spec: %v", err)
	}

	// Verify asyncapi version.
	if spec["asyncapi"] != "2.4.0" {
		t.Errorf("asyncapi = %v, want 2.4.0", spec["asyncapi"])
	}

	channels, ok := spec["channels"].(map[string]any)
	if !ok {
		t.Fatal("no channels in async spec")
	}
	ch, ok := channels["/ws"].(map[string]any)
	if !ok {
		t.Fatal("no /ws channel in async spec")
	}

	// subscribe = server→client = Send type
	sub, ok := ch["subscribe"].(map[string]any)
	if !ok {
		t.Fatal("no subscribe operation on /ws channel")
	}
	if sub["operationId"] == nil {
		t.Error("subscribe missing operationId")
	}
	if sub["message"] == nil {
		t.Error("subscribe missing message")
	}
	if sub["summary"] != "Echo WS" {
		t.Errorf("subscribe summary = %v, want Echo WS", sub["summary"])
	}

	// publish = client→server = Recv type
	pub, ok := ch["publish"].(map[string]any)
	if !ok {
		t.Fatal("no publish operation on /ws channel")
	}
	if pub["message"] == nil {
		t.Error("publish missing message")
	}
	if pub["summary"] != "Echo WS" {
		t.Errorf("publish summary = %v, want Echo WS", pub["summary"])
	}

	// Both operations should have tags.
	for _, opName := range []string{"subscribe", "publish"} {
		op := ch[opName].(map[string]any)
		tags, ok := op["tags"].([]any)
		if !ok || len(tags) == 0 {
			t.Errorf("%s missing tags", opName)
		} else {
			tag := tags[0].(map[string]any)
			if tag["name"] != "websocket" {
				t.Errorf("%s tag = %v, want websocket", opName, tag["name"])
			}
		}
	}

	// Verify schemas are in AsyncAPI components.
	components, ok := spec["components"].(map[string]any)
	if !ok {
		t.Fatal("no components in async spec")
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		t.Fatal("no schemas in async spec components")
	}
	if _, ok := schemas["wsServerMsg"]; !ok {
		t.Error("missing wsServerMsg schema in async spec")
	}
	if _, ok := schemas["wsClientMsg"]; !ok {
		t.Error("missing wsClientMsg schema in async spec")
	}

	// Verify WS path is NOT in OpenAPI spec.
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/openapi.json", nil)
	api.ServeHTTP(w2, r2)

	var oaSpec map[string]any
	if err := json.NewDecoder(w2.Body).Decode(&oaSpec); err != nil {
		t.Fatalf("decode openapi spec: %v", err)
	}
	if paths, ok := oaSpec["paths"].(map[string]any); ok {
		if _, ok := paths["/ws"]; ok {
			t.Error("WS path /ws should not be in OpenAPI spec")
		}
	}

	// Verify schemas are in OpenAPI components (for openapi-typescript).
	oaComponents, ok := oaSpec["components"].(map[string]any)
	if !ok {
		t.Fatal("no components in OpenAPI spec")
	}
	oaSchemas, ok := oaComponents["schemas"].(map[string]any)
	if !ok {
		t.Fatal("no schemas in OpenAPI components")
	}
	if _, ok := oaSchemas["wsServerMsg"]; !ok {
		t.Error("missing wsServerMsg schema in OpenAPI components")
	}
	if _, ok := oaSchemas["wsClientMsg"]; !ok {
		t.Error("missing wsClientMsg schema in OpenAPI components")
	}
}

func TestHandleWS_InputParsing(t *testing.T) {
	api := shiftapi.New()

	type Input struct {
		Channel string `query:"channel" validate:"required"`
	}

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket[Input](
			shiftapi.WSOn("msg", func(ctx context.Context, ws *shiftapi.WSSender, in Input, msg wsClientMsg) error {
				return ws.Send(ctx, wsServerMsg{Text: "channel=" + in.Channel})
			}),
		),
	)

	srv := httptest.NewServer(api)
	defer srv.Close()

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, srv.URL+"/ws?channel=general", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	// Send a message to trigger the handler.
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "msg", "data": map[string]any{"text": "hi"}}); err != nil {
		t.Fatalf("write: %v", err)
	}

	var msg wsServerMsg
	if err := wsjson.Read(ctx, conn, &msg); err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg.Text != "channel=general" {
		t.Errorf("got %q, want %q", msg.Text, "channel=general")
	}
	conn.Close(websocket.StatusNormalClosure, "")
}

func TestHandleWS_OnDispatch(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleWS(api, "GET /echo",
		shiftapi.Websocket[struct{}](
			shiftapi.WSOn("echo", func(ctx context.Context, ws *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
				return ws.Send(ctx, wsServerMsg{Text: "echo: " + msg.Text})
			}),
		),
	)

	srv := httptest.NewServer(api)
	defer srv.Close()

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, srv.URL+"/echo", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	// Send and receive multiple messages.
	for _, text := range []string{"hello", "world"} {
		envelope := map[string]any{"type": "echo", "data": map[string]any{"text": text}}
		if err := wsjson.Write(ctx, conn, envelope); err != nil {
			t.Fatalf("write %q: %v", text, err)
		}
		var resp wsServerMsg
		if err := wsjson.Read(ctx, conn, &resp); err != nil {
			t.Fatalf("read: %v", err)
		}
		want := "echo: " + text
		if resp.Text != want {
			t.Errorf("got %q, want %q", resp.Text, want)
		}
	}

	conn.Close(websocket.StatusNormalClosure, "")
}

func TestHandleWS_AutoWrapSend(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket[struct{}](
			shiftapi.WSOn("ping", func(ctx context.Context, ws *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
				return ws.Send(ctx, wsServerMsg{Text: "pong"})
			}),
			shiftapi.WSSends(
				shiftapi.MessageType[wsServerMsg]("server"),
			),
		),
	)

	srv := httptest.NewServer(api)
	defer srv.Close()

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, srv.URL+"/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	// Send a message
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "ping", "data": map[string]any{"text": "hi"}}); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read — should be wrapped in envelope {"type":"server","data":{...}}
	var envelope struct {
		Type string       `json:"type"`
		Data wsServerMsg  `json:"data"`
	}
	if err := wsjson.Read(ctx, conn, &envelope); err != nil {
		t.Fatalf("read: %v", err)
	}
	if envelope.Type != "server" {
		t.Errorf("envelope.Type = %q, want %q", envelope.Type, "server")
	}
	if envelope.Data.Text != "pong" {
		t.Errorf("envelope.Data.Text = %q, want %q", envelope.Data.Text, "pong")
	}
	conn.Close(websocket.StatusNormalClosure, "")
}

func TestHandleWS_ErrorBeforeUpgrade(t *testing.T) {
	api := shiftapi.New()

	type Input struct {
		Token string `query:"token" validate:"required"`
	}

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket[Input](
			shiftapi.WSOn("msg", func(ctx context.Context, ws *shiftapi.WSSender, in Input, msg wsClientMsg) error {
				return nil
			}),
		),
	)

	// Missing required query param → should get JSON error, not upgrade.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/ws", nil)
	api.ServeHTTP(w, r)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleWS_ErrorAfterUpgrade(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket[struct{}](
			shiftapi.WSOn("msg", func(ctx context.Context, ws *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
				return fmt.Errorf("something went wrong")
			}),
		),
	)

	srv := httptest.NewServer(api)
	defer srv.Close()

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, srv.URL+"/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	// Send a message to trigger the handler error.
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "msg", "data": map[string]any{"text": "hi"}}); err != nil {
		t.Fatalf("write: %v", err)
	}

	// The server should close the connection with StatusInternalError.
	_, _, err = conn.Read(ctx)
	if err == nil {
		t.Fatal("expected error from read")
	}
	status := websocket.CloseStatus(err)
	if status != websocket.StatusInternalError {
		t.Errorf("close status = %d, want %d", status, websocket.StatusInternalError)
	}
}

func TestHandleWS_WithWSAcceptOptions(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket[struct{}](
			shiftapi.WSOn("msg", func(ctx context.Context, ws *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
				return ws.Send(ctx, wsServerMsg{Text: "ok"})
			}),
		),
		shiftapi.WithWSAcceptOptions(shiftapi.WSAcceptOptions{
			Subprotocols: []string{"test-proto"},
		}),
	)

	srv := httptest.NewServer(api)
	defer srv.Close()

	ctx := context.Background()
	conn, resp, err := websocket.Dial(ctx, srv.URL+"/ws", &websocket.DialOptions{
		Subprotocols: []string{"test-proto"},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	// Verify the subprotocol was negotiated.
	if got := resp.Header.Get("Sec-WebSocket-Protocol"); got != "test-proto" {
		t.Errorf("subprotocol = %q, want %q", got, "test-proto")
	}

	// Send a message to trigger the handler.
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "msg", "data": map[string]any{"text": "hi"}}); err != nil {
		t.Fatalf("write: %v", err)
	}

	var msg wsServerMsg
	if err := wsjson.Read(ctx, conn, &msg); err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg.Text != "ok" {
		t.Errorf("got %q, want %q", msg.Text, "ok")
	}
	conn.Close(websocket.StatusNormalClosure, "")
}

func TestHandleWS_PathParams(t *testing.T) {
	api := shiftapi.New()

	type Input struct {
		ID string `path:"id"`
	}

	shiftapi.HandleWS(api, "GET /rooms/{id}",
		shiftapi.Websocket[Input](
			shiftapi.WSOn("msg", func(ctx context.Context, ws *shiftapi.WSSender, in Input, msg wsClientMsg) error {
				return ws.Send(ctx, wsServerMsg{Text: "room=" + in.ID})
			}),
		),
	)

	srv := httptest.NewServer(api)
	defer srv.Close()

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, srv.URL+"/rooms/abc", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	// Send a message to trigger the handler.
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "msg", "data": map[string]any{"text": "hi"}}); err != nil {
		t.Fatalf("write: %v", err)
	}

	var msg wsServerMsg
	if err := wsjson.Read(ctx, conn, &msg); err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg.Text != "room=abc" {
		t.Errorf("got %q, want %q", msg.Text, "room=abc")
	}
	conn.Close(websocket.StatusNormalClosure, "")
}

// --- Multi-message (WSSends) tests ---

type wsChatMsg struct {
	User string `json:"user"`
	Text string `json:"text"`
}

type wsSystemMsg struct {
	Info string `json:"info"`
}

type wsUserMsg struct {
	Text string `json:"text"`
}

type wsUserCmd struct {
	Command string `json:"command"`
}

func TestHandleWS_MultiTypeDispatch(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket[struct{}](
			shiftapi.WSOn("message", func(ctx context.Context, ws *shiftapi.WSSender, _ struct{}, m wsUserMsg) error {
				return ws.Send(ctx, wsChatMsg{User: "server", Text: "got: " + m.Text})
			}),
			shiftapi.WSOn("command", func(ctx context.Context, ws *shiftapi.WSSender, _ struct{}, cmd wsUserCmd) error {
				return ws.Send(ctx, wsSystemMsg{Info: "executed: " + cmd.Command})
			}),
			shiftapi.WSSends(
				shiftapi.MessageType[wsChatMsg]("chat"),
				shiftapi.MessageType[wsSystemMsg]("system"),
			),
		),
	)

	srv := httptest.NewServer(api)
	defer srv.Close()

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, srv.URL+"/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	// Send a "message" type
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "message", "data": map[string]any{"text": "hello"}}); err != nil {
		t.Fatalf("write: %v", err)
	}

	var msg1 struct {
		Type string     `json:"type"`
		Data wsChatMsg  `json:"data"`
	}
	if err := wsjson.Read(ctx, conn, &msg1); err != nil {
		t.Fatalf("read 1: %v", err)
	}
	if msg1.Type != "chat" {
		t.Errorf("msg1.Type = %q, want %q", msg1.Type, "chat")
	}
	if msg1.Data.Text != "got: hello" {
		t.Errorf("msg1.Data.Text = %q, want %q", msg1.Data.Text, "got: hello")
	}

	// Send a "command" type
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "command", "data": map[string]any{"command": "quit"}}); err != nil {
		t.Fatalf("write: %v", err)
	}

	var msg2 struct {
		Type string       `json:"type"`
		Data wsSystemMsg  `json:"data"`
	}
	if err := wsjson.Read(ctx, conn, &msg2); err != nil {
		t.Fatalf("read 2: %v", err)
	}
	if msg2.Type != "system" {
		t.Errorf("msg2.Type = %q, want %q", msg2.Type, "system")
	}
	if msg2.Data.Info != "executed: quit" {
		t.Errorf("msg2.Data.Info = %q, want %q", msg2.Data.Info, "executed: quit")
	}

	conn.Close(websocket.StatusNormalClosure, "")
}

func TestHandleWS_WithMessages_AsyncAPISpec(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket[struct{}](
			shiftapi.WSOn("message", func(ctx context.Context, ws *shiftapi.WSSender, _ struct{}, m wsUserMsg) error {
				return nil
			}),
			shiftapi.WSOn("command", func(ctx context.Context, ws *shiftapi.WSSender, _ struct{}, cmd wsUserCmd) error {
				return nil
			}),
			shiftapi.WSSends(
				shiftapi.MessageType[wsChatMsg]("chat"),
				shiftapi.MessageType[wsSystemMsg]("system"),
			),
		),
	)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/asyncapi.json", nil)
	api.ServeHTTP(w, r)

	var spec map[string]any
	if err := json.NewDecoder(w.Body).Decode(&spec); err != nil {
		t.Fatalf("decode spec: %v", err)
	}

	channels := spec["channels"].(map[string]any)
	ch := channels["/ws"].(map[string]any)

	// subscribe = server→client = Send with oneOf variants
	sub := ch["subscribe"].(map[string]any)
	subMsg := sub["message"].(map[string]any)
	subOneOf, ok := subMsg["oneOf"].([]any)
	if !ok {
		t.Fatal("subscribe message missing oneOf")
	}
	if len(subOneOf) != 2 {
		t.Fatalf("subscribe oneOf has %d items, want 2", len(subOneOf))
	}

	// publish = client→server = Recv with oneOf variants (from On handlers)
	pub := ch["publish"].(map[string]any)
	pubMsg := pub["message"].(map[string]any)
	pubOneOf, ok := pubMsg["oneOf"].([]any)
	if !ok {
		t.Fatal("publish message missing oneOf")
	}
	if len(pubOneOf) != 2 {
		t.Fatalf("publish oneOf has %d items, want 2", len(pubOneOf))
	}

	// Verify envelope schemas exist in components.
	components := spec["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)

	// Payload schemas
	if _, ok := schemas["wsChatMsg"]; !ok {
		t.Error("missing wsChatMsg schema")
	}
	if _, ok := schemas["wsSystemMsg"]; !ok {
		t.Error("missing wsSystemMsg schema")
	}
	if _, ok := schemas["wsUserMsg"]; !ok {
		t.Error("missing wsUserMsg schema")
	}
	if _, ok := schemas["wsUserCmd"]; !ok {
		t.Error("missing wsUserCmd schema")
	}

	// Envelope schemas
	if _, ok := schemas["chat_wsChatMsg"]; !ok {
		t.Error("missing chat_wsChatMsg envelope schema")
	}
	if _, ok := schemas["system_wsSystemMsg"]; !ok {
		t.Error("missing system_wsSystemMsg envelope schema")
	}
}

func TestMessageType_EmptyNamePanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for empty message name")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "must not be empty") {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()
	shiftapi.MessageType[wsClientMsg]("")
}

func TestHandleWS_DuplicateSendNamePanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for duplicate message name")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "duplicate message name") {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()
	api := shiftapi.New()
	shiftapi.HandleWS(api, "GET /dup",
		shiftapi.Websocket[struct{}](
			shiftapi.WSOn("msg", func(ctx context.Context, ws *shiftapi.WSSender, _ struct{}, m wsClientMsg) error {
				return nil
			}),
			shiftapi.WSSends(
				shiftapi.MessageType[wsChatMsg]("same"),
				shiftapi.MessageType[wsSystemMsg]("same"),
			),
		),
	)
}

func TestHandleWS_DuplicateOnNamePanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for duplicate On name")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "duplicate message name") {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()
	api := shiftapi.New()
	shiftapi.HandleWS(api, "GET /dup",
		shiftapi.Websocket[struct{}](
			shiftapi.WSOn("msg", func(ctx context.Context, ws *shiftapi.WSSender, _ struct{}, m wsClientMsg) error {
				return nil
			}),
			shiftapi.WSOn("msg", func(ctx context.Context, ws *shiftapi.WSSender, _ struct{}, m wsUserMsg) error {
				return nil
			}),
		),
	)
}

func TestOn_EmptyNamePanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for empty On name")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "must not be empty") {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()
	shiftapi.WSOn("", func(ctx context.Context, ws *shiftapi.WSSender, _ struct{}, m wsClientMsg) error {
		return nil
	})
}

func TestWebsocket_NoHandlersPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for no On handlers")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "at least one WSOn handler") {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()
	shiftapi.Websocket[struct{}](
		shiftapi.WSSends(
			shiftapi.MessageType[wsServerMsg]("server"),
		),
	)
}
