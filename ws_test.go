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

func noSetup(r *http.Request, sender *shiftapi.WSSender, _ struct{}) (struct{}, error) {
	return struct{}{}, nil
}

// wsErrorFrame represents the wire format for error frames:
// {"error": true, "code": 4xxx, "data": ...}
type wsErrorFrame struct {
	Error bool            `json:"error"`
	Code  int             `json:"code"`
	Data  json.RawMessage `json:"data"`
}

// readWSError reads an error frame envelope and decodes its data field into v.
func readWSError(t *testing.T, ctx context.Context, conn *websocket.Conn, v any) wsErrorFrame {
	t.Helper()
	var frame wsErrorFrame
	if err := wsjson.Read(ctx, conn, &frame); err != nil {
		t.Fatalf("read error frame: %v", err)
	}
	if !frame.Error {
		t.Fatal("expected error frame (error: true)")
	}
	if v != nil {
		if err := json.Unmarshal(frame.Data, v); err != nil {
			t.Fatalf("unmarshal error data: %v", err)
		}
	}
	return frame
}

func TestHandleWS_AsyncAPISpec(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket(
			noSetup,
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
			shiftapi.WSOn("echo", func(sender *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
				return sender.Send(wsServerMsg(msg))
			}),
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

func TestHandleWS_AsyncAPISpec_XErrors(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket(
			func(r *http.Request, sender *shiftapi.WSSender, _ struct{}) (struct{}, error) {
				return struct{}{}, nil
			},
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
			shiftapi.WSOn("msg", func(sender *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
				return nil
			}),
		),
		shiftapi.WithError[*wsAuthError](http.StatusUnauthorized),
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

	xErrors, ok := ch["x-errors"].(map[string]any)
	if !ok {
		t.Fatal("no x-errors on /ws channel")
	}

	// Should have 4401 for wsAuthError and 4422 for ValidationError.
	if _, ok := xErrors["4401"]; !ok {
		t.Error("missing 4401 in x-errors")
	}
	if _, ok := xErrors["4422"]; !ok {
		t.Error("missing 4422 in x-errors")
	}

	// Verify the 4401 entry references wsAuthError schema.
	entry := xErrors["4401"].(map[string]any)
	ref, _ := entry["$ref"].(string)
	if ref != "#/components/schemas/wsAuthError" {
		t.Errorf("4401 $ref = %q, want #/components/schemas/wsAuthError", ref)
	}

	// Verify the schema is registered in both specs.
	components := spec["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	if _, ok := schemas["wsAuthError"]; !ok {
		t.Error("missing wsAuthError schema in AsyncAPI components")
	}
}

func TestHandleWS_InputParsing(t *testing.T) {
	api := shiftapi.New()

	type Input struct {
		Channel string `query:"channel" validate:"required"`
	}

	type inputState struct {
		Channel string
	}

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket(
			func(r *http.Request, sender *shiftapi.WSSender, in Input) (*inputState, error) {
				return &inputState{Channel: in.Channel}, nil
			},
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
			shiftapi.WSOn("msg", func(sender *shiftapi.WSSender, state *inputState, msg wsClientMsg) error {
				return sender.Send(wsServerMsg{Text: "channel=" + state.Channel})
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
	defer conn.CloseNow() //nolint:errcheck

	// Send a message to trigger the handler.
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "msg", "data": map[string]any{"text": "hi"}}); err != nil {
		t.Fatalf("write: %v", err)
	}

	var envelope struct {
		Type string      `json:"type"`
		Data wsServerMsg `json:"data"`
	}
	if err := wsjson.Read(ctx, conn, &envelope); err != nil {
		t.Fatalf("read: %v", err)
	}
	if envelope.Data.Text != "channel=general" {
		t.Errorf("got %q, want %q", envelope.Data.Text, "channel=general")
	}
	conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck
}

func TestHandleWS_OnDispatch(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleWS(api, "GET /echo",
		shiftapi.Websocket(
			noSetup,
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
			shiftapi.WSOn("echo", func(sender *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
				return sender.Send(wsServerMsg{Text: "echo: " + msg.Text})
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
	defer conn.CloseNow() //nolint:errcheck

	// Send and receive multiple messages.
	for _, text := range []string{"hello", "world"} {
		envelope := map[string]any{"type": "echo", "data": map[string]any{"text": text}}
		if err := wsjson.Write(ctx, conn, envelope); err != nil {
			t.Fatalf("write %q: %v", text, err)
		}
		var resp struct {
			Type string      `json:"type"`
			Data wsServerMsg `json:"data"`
		}
		if err := wsjson.Read(ctx, conn, &resp); err != nil {
			t.Fatalf("read: %v", err)
		}
		want := "echo: " + text
		if resp.Data.Text != want {
			t.Errorf("got %q, want %q", resp.Data.Text, want)
		}
	}

	conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck
}

func TestHandleWS_AutoWrapSend(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket(
			noSetup,
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
			shiftapi.WSOn("ping", func(sender *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
				return sender.Send(wsServerMsg{Text: "pong"})
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
	defer conn.CloseNow() //nolint:errcheck

	// Send a message
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "ping", "data": map[string]any{"text": "hi"}}); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read — should be wrapped in envelope {"type":"server","data":{...}}
	var envelope struct {
		Type string      `json:"type"`
		Data wsServerMsg `json:"data"`
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
	conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck
}

func TestHandleWS_ErrorBeforeUpgrade(t *testing.T) {
	api := shiftapi.New()

	type Input struct {
		Token string `query:"token" validate:"required"`
	}

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket(
			func(r *http.Request, sender *shiftapi.WSSender, in Input) (struct{}, error) {
				return struct{}{}, nil
			},
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
			shiftapi.WSOn("msg", func(sender *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
				return nil
			}),
		),
	)

	srv := httptest.NewServer(api)
	defer srv.Close()

	ctx := context.Background()
	// Missing required query param → connection opens, error sent as first frame.
	conn, _, err := websocket.Dial(ctx, srv.URL+"/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow() //nolint:errcheck

	// First frame should be a structured error envelope.
	var errResp shiftapi.ValidationError
	frame := readWSError(t, ctx, conn, &errResp)
	if frame.Code != 4422 {
		t.Errorf("error frame code = %d, want 4422", frame.Code)
	}
	if errResp.Message != "validation failed" {
		t.Errorf("message = %q, want %q", errResp.Message, "validation failed")
	}

	// Connection should close with 4422.
	_, _, err = conn.Read(ctx)
	if websocket.CloseStatus(err) != 4422 {
		t.Errorf("close code = %d, want 4422", websocket.CloseStatus(err))
	}
}

func TestHandleWS_ErrorAfterUpgrade(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket(
			noSetup,
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
			shiftapi.WSOn("msg", func(sender *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
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
	defer conn.CloseNow() //nolint:errcheck

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

func TestHandleWS_ErrorAfterUpgrade_Registered(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket(
			noSetup,
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
			shiftapi.WSOn("msg", func(sender *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
				return &wsAuthError{Message: "token expired", Realm: "api"}
			}),
		),
		shiftapi.WithError[*wsAuthError](http.StatusUnauthorized),
	)

	srv := httptest.NewServer(api)
	defer srv.Close()

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, srv.URL+"/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow() //nolint:errcheck

	// Send a message to trigger the handler error.
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "msg", "data": map[string]any{"text": "hi"}}); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Should receive a structured error envelope, not a data frame.
	var errResp wsAuthError
	frame := readWSError(t, ctx, conn, &errResp)
	if frame.Code != 4401 {
		t.Errorf("error frame code = %d, want 4401", frame.Code)
	}
	if errResp.Message != "token expired" {
		t.Errorf("message = %q, want %q", errResp.Message, "token expired")
	}
	if errResp.Realm != "api" {
		t.Errorf("realm = %q, want %q", errResp.Realm, "api")
	}

	// Connection should close with 4401.
	_, _, err = conn.Read(ctx)
	if websocket.CloseStatus(err) != 4401 {
		t.Errorf("close code = %d, want 4401", websocket.CloseStatus(err))
	}
}

// wsAuthError is an error type registered via WithError for setup error tests.
type wsAuthError struct {
	Message string `json:"message"`
	Realm   string `json:"realm"`
}

func (e *wsAuthError) Error() string { return e.Message }

func TestHandleWS_SetupErrorRegistered(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket(
			func(r *http.Request, sender *shiftapi.WSSender, _ struct{}) (struct{}, error) {
				return struct{}{}, &wsAuthError{Message: "bad token", Realm: "api"}
			},
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
			shiftapi.WSOn("msg", func(sender *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
				return nil
			}),
		),
		shiftapi.WithError[*wsAuthError](http.StatusUnauthorized),
	)

	srv := httptest.NewServer(api)
	defer srv.Close()

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, srv.URL+"/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow() //nolint:errcheck

	// First frame should be a structured error envelope.
	var errResp wsAuthError
	frame := readWSError(t, ctx, conn, &errResp)
	if frame.Code != 4401 {
		t.Errorf("error frame code = %d, want 4401", frame.Code)
	}
	if errResp.Message != "bad token" {
		t.Errorf("message = %q, want %q", errResp.Message, "bad token")
	}
	if errResp.Realm != "api" {
		t.Errorf("realm = %q, want %q", errResp.Realm, "api")
	}

	// Connection should close with 4401.
	_, _, err = conn.Read(ctx)
	if websocket.CloseStatus(err) != 4401 {
		t.Errorf("close code = %d, want 4401", websocket.CloseStatus(err))
	}
}

func TestHandleWS_SetupErrorUnregistered(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket(
			func(r *http.Request, sender *shiftapi.WSSender, _ struct{}) (struct{}, error) {
				return struct{}{}, fmt.Errorf("unexpected failure")
			},
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
			shiftapi.WSOn("msg", func(sender *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
				return nil
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
	defer conn.CloseNow() //nolint:errcheck

	// Unregistered error → no error frame, just StatusInternalError close.
	_, _, err = conn.Read(ctx)
	if err == nil {
		t.Fatal("expected error from read")
	}
	if websocket.CloseStatus(err) != websocket.StatusInternalError {
		t.Errorf("close status = %d, want %d", websocket.CloseStatus(err), websocket.StatusInternalError)
	}
}

func TestHandleWS_SetupValidationError(t *testing.T) {
	api := shiftapi.New()

	type SetupInput struct {
		Code string `query:"code"`
	}

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket(
			func(r *http.Request, sender *shiftapi.WSSender, in SetupInput) (struct{}, error) {
				if in.Code != "secret" {
					return struct{}{}, &shiftapi.ValidationError{
						Message: "validation failed",
						Errors: []shiftapi.FieldError{{Field: "code", Message: "invalid code"}},
					}
				}
				return struct{}{}, nil
			},
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
			shiftapi.WSOn("msg", func(sender *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
				return nil
			}),
		),
	)

	srv := httptest.NewServer(api)
	defer srv.Close()

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, srv.URL+"/ws?code=wrong", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow() //nolint:errcheck

	// ValidationError is always matched → sent as error envelope with 4422.
	var errResp shiftapi.ValidationError
	frame := readWSError(t, ctx, conn, &errResp)
	if frame.Code != 4422 {
		t.Errorf("error frame code = %d, want 4422", frame.Code)
	}
	if errResp.Message != "validation failed" {
		t.Errorf("message = %q, want %q", errResp.Message, "validation failed")
	}
	if len(errResp.Errors) != 1 || errResp.Errors[0].Field != "code" {
		t.Errorf("field errors = %v, want [{code invalid code}]", errResp.Errors)
	}

	_, _, err = conn.Read(ctx)
	if websocket.CloseStatus(err) != 4422 {
		t.Errorf("close code = %d, want 4422", websocket.CloseStatus(err))
	}
}

func TestHandleWS_WSOnUnknownMessage(t *testing.T) {
	api := shiftapi.New()

	var gotType string
	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket(
			noSetup,
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
			shiftapi.WSOn("msg", func(sender *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
				return sender.Send(wsServerMsg{Text: "ok"})
			}),
			shiftapi.WSOnUnknownMessage(func(sender *shiftapi.WSSender, _ struct{}, msgType string, data json.RawMessage) {
				gotType = msgType
				sender.Send(wsServerMsg{Text: "unknown: " + msgType}) //nolint:errcheck
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
	defer conn.CloseNow() //nolint:errcheck

	// Send an unknown message type.
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "bogus", "data": map[string]any{"text": "hi"}}); err != nil {
		t.Fatalf("write: %v", err)
	}

	// The callback should have sent a response.
	var envelope struct {
		Type string      `json:"type"`
		Data wsServerMsg `json:"data"`
	}
	if err := wsjson.Read(ctx, conn, &envelope); err != nil {
		t.Fatalf("read: %v", err)
	}
	if envelope.Data.Text != "unknown: bogus" {
		t.Errorf("got %q, want %q", envelope.Data.Text, "unknown: bogus")
	}
	if gotType != "bogus" {
		t.Errorf("gotType = %q, want %q", gotType, "bogus")
	}
	conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck
}

func TestHandleWS_WithWSAcceptOptions(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket(
			noSetup,
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
			shiftapi.WSOn("msg", func(sender *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
				return sender.Send(wsServerMsg{Text: "ok"})
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
	defer conn.CloseNow() //nolint:errcheck

	// Verify the subprotocol was negotiated.
	if got := resp.Header.Get("Sec-WebSocket-Protocol"); got != "test-proto" {
		t.Errorf("subprotocol = %q, want %q", got, "test-proto")
	}

	// Send a message to trigger the handler.
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "msg", "data": map[string]any{"text": "hi"}}); err != nil {
		t.Fatalf("write: %v", err)
	}

	var envelope struct {
		Type string      `json:"type"`
		Data wsServerMsg `json:"data"`
	}
	if err := wsjson.Read(ctx, conn, &envelope); err != nil {
		t.Fatalf("read: %v", err)
	}
	if envelope.Data.Text != "ok" {
		t.Errorf("got %q, want %q", envelope.Data.Text, "ok")
	}
	conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck
}

func TestHandleWS_PathParams(t *testing.T) {
	api := shiftapi.New()

	type Input struct {
		ID string `path:"id"`
	}

	type pathState struct {
		ID string
	}

	shiftapi.HandleWS(api, "GET /rooms/{id}",
		shiftapi.Websocket(
			func(r *http.Request, sender *shiftapi.WSSender, in Input) (*pathState, error) {
				return &pathState{ID: in.ID}, nil
			},
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
			shiftapi.WSOn("msg", func(sender *shiftapi.WSSender, state *pathState, msg wsClientMsg) error {
				return sender.Send(wsServerMsg{Text: "room=" + state.ID})
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
	defer conn.CloseNow() //nolint:errcheck

	// Send a message to trigger the handler.
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "msg", "data": map[string]any{"text": "hi"}}); err != nil {
		t.Fatalf("write: %v", err)
	}

	var envelope struct {
		Type string      `json:"type"`
		Data wsServerMsg `json:"data"`
	}
	if err := wsjson.Read(ctx, conn, &envelope); err != nil {
		t.Fatalf("read: %v", err)
	}
	if envelope.Data.Text != "room=abc" {
		t.Errorf("got %q, want %q", envelope.Data.Text, "room=abc")
	}
	conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck
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
		shiftapi.Websocket(
			noSetup,
			shiftapi.WSSends(
				shiftapi.WSMessageType[wsChatMsg]("chat"),
				shiftapi.WSMessageType[wsSystemMsg]("system"),
			),
			shiftapi.WSOn("message", func(sender *shiftapi.WSSender, _ struct{}, m wsUserMsg) error {
				return sender.Send(wsChatMsg{User: "server", Text: "got: " + m.Text})
			}),
			shiftapi.WSOn("command", func(sender *shiftapi.WSSender, _ struct{}, cmd wsUserCmd) error {
				return sender.Send(wsSystemMsg{Info: "executed: " + cmd.Command})
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
	defer conn.CloseNow() //nolint:errcheck

	// Send a "message" type
	if err := wsjson.Write(ctx, conn, map[string]any{"type": "message", "data": map[string]any{"text": "hello"}}); err != nil {
		t.Fatalf("write: %v", err)
	}

	var msg1 struct {
		Type string    `json:"type"`
		Data wsChatMsg `json:"data"`
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
		Type string      `json:"type"`
		Data wsSystemMsg `json:"data"`
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

	conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck
}

func TestHandleWS_WithMessages_AsyncAPISpec(t *testing.T) {
	api := shiftapi.New()

	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket(
			noSetup,
			shiftapi.WSSends(
				shiftapi.WSMessageType[wsChatMsg]("chat"),
				shiftapi.WSMessageType[wsSystemMsg]("system"),
			),
			shiftapi.WSOn("message", func(sender *shiftapi.WSSender, _ struct{}, m wsUserMsg) error {
				return nil
			}),
			shiftapi.WSOn("command", func(sender *shiftapi.WSSender, _ struct{}, cmd wsUserCmd) error {
				return nil
			}),
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
	shiftapi.WSMessageType[wsClientMsg]("")
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
		shiftapi.Websocket(
			noSetup,
			shiftapi.WSSends(
				shiftapi.WSMessageType[wsChatMsg]("same"),
				shiftapi.WSMessageType[wsSystemMsg]("same"),
			),
			shiftapi.WSOn("msg", func(sender *shiftapi.WSSender, _ struct{}, m wsClientMsg) error {
				return nil
			}),
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
		shiftapi.Websocket(
			noSetup,
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
			shiftapi.WSOn("msg", func(sender *shiftapi.WSSender, _ struct{}, m wsClientMsg) error {
				return nil
			}),
			shiftapi.WSOn("msg", func(sender *shiftapi.WSSender, _ struct{}, m wsUserMsg) error {
				return nil
			}),
		),
	)
}

func TestHandleWS_Setup(t *testing.T) {
	api := shiftapi.New()

	type joinInput struct {
		Room string `query:"room"`
	}

	type roomState struct {
		Room string
	}

	shiftapi.HandleWS(api, "GET /chat",
		shiftapi.Websocket(
			func(r *http.Request, sender *shiftapi.WSSender, in joinInput) (*roomState, error) {
				if in.Room == "" {
					return nil, fmt.Errorf("room required")
				}
				return &roomState{Room: in.Room}, nil
			},
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
			shiftapi.WSOn("message", func(sender *shiftapi.WSSender, state *roomState, msg wsClientMsg) error {
				return sender.Send(wsServerMsg{Text: "[" + state.Room + "] " + msg.Text})
			}),
		),
	)

	srv := httptest.NewServer(api)
	defer srv.Close()

	ctx := context.Background()

	// Test successful setup — query param available to handlers.
	conn, _, err := websocket.Dial(ctx, srv.URL+"/chat?room=general", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow() //nolint:errcheck

	envelope := map[string]any{"type": "message", "data": map[string]any{"text": "hello"}}
	if err := wsjson.Write(ctx, conn, envelope); err != nil {
		t.Fatalf("write: %v", err)
	}

	var resp struct {
		Type string      `json:"type"`
		Data wsServerMsg `json:"data"`
	}
	if err := wsjson.Read(ctx, conn, &resp); err != nil {
		t.Fatalf("read: %v", err)
	}
	if resp.Data.Text != "[general] hello" {
		t.Errorf("got %q, want %q", resp.Data.Text, "[general] hello")
	}
	conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck
}

func TestHandleWS_Setup_Error(t *testing.T) {
	api := shiftapi.New()

	type joinInput struct {
		Room string `query:"room"`
	}

	shiftapi.HandleWS(api, "GET /chat",
		shiftapi.Websocket(
			func(r *http.Request, sender *shiftapi.WSSender, in joinInput) (struct{}, error) {
				return struct{}{}, fmt.Errorf("setup failed")
			},
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
			shiftapi.WSOn("message", func(sender *shiftapi.WSSender, _ struct{}, msg wsClientMsg) error {
				return sender.Send(wsServerMsg{Text: "should not reach"})
			}),
		),
	)

	srv := httptest.NewServer(api)
	defer srv.Close()

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, srv.URL+"/chat?room=general", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow() //nolint:errcheck

	// The connection should be closed by the server due to setup error.
	var msg json.RawMessage
	err = wsjson.Read(ctx, conn, &msg)
	if err == nil {
		t.Fatal("expected error reading from connection closed by setup failure")
	}
	if websocket.CloseStatus(err) != websocket.StatusInternalError {
		t.Errorf("close status = %d, want %d (StatusInternalError)", websocket.CloseStatus(err), websocket.StatusInternalError)
	}
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
	shiftapi.WSOn("", func(sender *shiftapi.WSSender, _ struct{}, m wsClientMsg) error {
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
	api := shiftapi.New()
	shiftapi.HandleWS(api, "GET /ws",
		shiftapi.Websocket(
			noSetup,
			shiftapi.WSSends(shiftapi.WSMessageType[wsServerMsg]("server")),
		),
	)
}
