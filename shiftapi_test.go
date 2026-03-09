package shiftapi_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"slices"
	"strings"
	"testing"

	"github.com/fcjr/shiftapi"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-playground/validator/v10"
)

// --- Test types ---

type Person struct {
	Name string `json:"name"`
}

type Greeting struct {
	Hello string `json:"hello"`
}

type Status struct {
	OK bool `json:"ok"`
}

type Item struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Empty struct{}

// --- Helpers ---

func newTestAPI(t *testing.T) *shiftapi.API {
	t.Helper()
	return shiftapi.New()
}

func doRequest(t *testing.T, api http.Handler, method, path string, body string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	return rec.Result()
}

func decodeJSON[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	var v T
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return v
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	return string(b)
}

// --- API creation tests ---

func TestNew(t *testing.T) {
	api := shiftapi.New()
	if api == nil {
		t.Fatal("New() returned nil")
	}
}

func TestNewWithOptions(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithInfo(shiftapi.Info{
			Title:       "Test API",
			Description: "A test API",
			Version:     "1.0.0",
		}),
	)
	spec := api.Spec()
	if spec.Info == nil {
		t.Fatal("expected spec.Info to be set")
	}
	if spec.Info.Title != "Test API" {
		t.Errorf("expected title %q, got %q", "Test API", spec.Info.Title)
	}
	if spec.Info.Description != "A test API" {
		t.Errorf("expected description %q, got %q", "A test API", spec.Info.Description)
	}
	if spec.Info.Version != "1.0.0" {
		t.Errorf("expected version %q, got %q", "1.0.0", spec.Info.Version)
	}
}

func TestWithInfoContact(t *testing.T) {
	api := shiftapi.New(shiftapi.WithInfo(shiftapi.Info{
		Title: "Test",
		Contact: &shiftapi.Contact{
			Name:  "Dev",
			URL:   "https://example.com",
			Email: "dev@example.com",
		},
	}))
	spec := api.Spec()
	if spec.Info.Contact == nil {
		t.Fatal("expected contact to be set")
	}
	if spec.Info.Contact.Name != "Dev" {
		t.Errorf("expected contact name %q, got %q", "Dev", spec.Info.Contact.Name)
	}
	if spec.Info.Contact.URL != "https://example.com" {
		t.Errorf("expected contact URL %q, got %q", "https://example.com", spec.Info.Contact.URL)
	}
	if spec.Info.Contact.Email != "dev@example.com" {
		t.Errorf("expected contact email %q, got %q", "dev@example.com", spec.Info.Contact.Email)
	}
}

func TestWithInfoLicense(t *testing.T) {
	api := shiftapi.New(shiftapi.WithInfo(shiftapi.Info{
		Title: "Test",
		License: &shiftapi.License{
			Name: "MIT",
			URL:  "https://opensource.org/licenses/MIT",
		},
	}))
	spec := api.Spec()
	if spec.Info.License == nil {
		t.Fatal("expected license to be set")
	}
	if spec.Info.License.Name != "MIT" {
		t.Errorf("expected license name %q, got %q", "MIT", spec.Info.License.Name)
	}
}

func TestWithExternalDocs(t *testing.T) {
	api := shiftapi.New(shiftapi.WithExternalDocs(shiftapi.ExternalDocs{
		Description: "More info",
		URL:         "https://example.com/docs",
	}))
	spec := api.Spec()
	if spec.ExternalDocs == nil {
		t.Fatal("expected ExternalDocs to be set")
	}
	if spec.ExternalDocs.Description != "More info" {
		t.Errorf("expected description %q, got %q", "More info", spec.ExternalDocs.Description)
	}
	if spec.ExternalDocs.URL != "https://example.com/docs" {
		t.Errorf("expected URL %q, got %q", "https://example.com/docs", spec.ExternalDocs.URL)
	}
}

// --- Built-in endpoint tests ---

func TestServeOpenAPISpec(t *testing.T) {
	api := shiftapi.New(shiftapi.WithInfo(shiftapi.Info{
		Title:   "Spec Test",
		Version: "2.0",
	}))
	shiftapi.Get(api, "/health", func(r *http.Request, _ struct{}) (*Status, error) {
		return &Status{OK: true}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/openapi.json", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type application/json; charset=utf-8, got %q", ct)
	}

	var spec map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
		t.Fatalf("failed to decode spec: %v", err)
	}
	_ = resp.Body.Close()

	if spec["openapi"] != "3.1" {
		t.Errorf("expected openapi 3.1, got %v", spec["openapi"])
	}
	info, ok := spec["info"].(map[string]any)
	if !ok {
		t.Fatal("expected info in spec")
	}
	if info["title"] != "Spec Test" {
		t.Errorf("expected title %q, got %v", "Spec Test", info["title"])
	}
}

func TestServeDocs(t *testing.T) {
	api := shiftapi.New(shiftapi.WithInfo(shiftapi.Info{Title: "Docs Test"}))
	resp := doRequest(t, api, http.MethodGet, "/docs", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body := readBody(t, resp)
	if !strings.Contains(body, "Scalar") {
		t.Error("expected docs page to contain 'Scalar'")
	}
	if !strings.Contains(body, "Docs Test") {
		t.Error("expected docs page to contain the API title")
	}
}

func TestRootRedirectsToDocs(t *testing.T) {
	api := shiftapi.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	resp := rec.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if loc != "/docs" {
		t.Errorf("expected redirect to /docs, got %q", loc)
	}
}

// --- POST handler tests ---

func TestPostHandler(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/greet", func(r *http.Request, in *Person) (*Greeting, error) {
		return &Greeting{Hello: in.Name}, nil
	})

	resp := doRequest(t, api, http.MethodPost, "/greet", `{"name":"alice"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	greeting := decodeJSON[Greeting](t, resp)
	if greeting.Hello != "alice" {
		t.Errorf("expected Hello=alice, got %q", greeting.Hello)
	}
}

func TestPostHandlerInvalidBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/greet", func(r *http.Request, in *Person) (*Greeting, error) {
		return &Greeting{Hello: in.Name}, nil
	})

	resp := doRequest(t, api, http.MethodPost, "/greet", `not json`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPostHandlerEmptyBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/greet", func(r *http.Request, in *Person) (*Greeting, error) {
		return &Greeting{Hello: in.Name}, nil
	})

	resp := doRequest(t, api, http.MethodPost, "/greet", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPostHandlerEmptyJSONObject(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/person", func(r *http.Request, in *ValidatedPerson) (*ValidatedPerson, error) {
		return in, nil
	})

	resp := doRequest(t, api, http.MethodPost, "/person", `{}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

// --- GET handler tests ---

func TestGetHandler(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/health", func(r *http.Request, _ struct{}) (*Status, error) {
		return &Status{OK: true}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/health", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	status := decodeJSON[Status](t, resp)
	if !status.OK {
		t.Error("expected OK=true")
	}
}

func TestGetHandlerWithPathParam(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/items/{id}", func(r *http.Request, _ struct{}) (*Item, error) {
		return &Item{ID: r.PathValue("id"), Name: "widget"}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/items/abc123", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	item := decodeJSON[Item](t, resp)
	if item.ID != "abc123" {
		t.Errorf("expected ID=abc123, got %q", item.ID)
	}
	if item.Name != "widget" {
		t.Errorf("expected Name=widget, got %q", item.Name)
	}
}

// --- PUT handler tests ---

func TestPutHandler(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Put(api, "/items/{id}", func(r *http.Request, in *Item) (*Item, error) {
		in.ID = r.PathValue("id")
		return in, nil
	})

	resp := doRequest(t, api, http.MethodPut, "/items/42", `{"name":"updated"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	item := decodeJSON[Item](t, resp)
	if item.ID != "42" {
		t.Errorf("expected ID=42, got %q", item.ID)
	}
	if item.Name != "updated" {
		t.Errorf("expected Name=updated, got %q", item.Name)
	}
}

// --- PATCH handler tests ---

func TestPatchHandler(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Patch(api, "/items/{id}", func(r *http.Request, in *Item) (*Item, error) {
		in.ID = r.PathValue("id")
		return in, nil
	})

	resp := doRequest(t, api, http.MethodPatch, "/items/99", `{"name":"patched"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	item := decodeJSON[Item](t, resp)
	if item.Name != "patched" {
		t.Errorf("expected Name=patched, got %q", item.Name)
	}
}

// --- DELETE handler tests ---

func TestDeleteHandler(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Delete(api, "/items/{id}", func(r *http.Request, _ struct{}) (*Empty, error) {
		return &Empty{}, nil
	})

	resp := doRequest(t, api, http.MethodDelete, "/items/42", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// --- HEAD handler tests ---

func TestHeadHandler(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Head(api, "/ping", func(r *http.Request, _ struct{}) (*Empty, error) {
		return &Empty{}, nil
	})

	resp := doRequest(t, api, http.MethodHead, "/ping", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// --- OPTIONS handler tests ---

func TestOptionsHandler(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Options(api, "/items", func(r *http.Request, _ struct{}) (*Empty, error) {
		return &Empty{}, nil
	})

	resp := doRequest(t, api, http.MethodOptions, "/items", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// --- TRACE handler tests ---

func TestTraceHandler(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Trace(api, "/debug", func(r *http.Request, _ struct{}) (*Empty, error) {
		return &Empty{}, nil
	})

	resp := doRequest(t, api, http.MethodTrace, "/debug", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// --- CONNECT handler tests ---

func TestConnectHandler(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Connect(api, "/tunnel", func(r *http.Request, _ struct{}) (*Empty, error) {
		return &Empty{}, nil
	})

	resp := doRequest(t, api, http.MethodConnect, "/tunnel", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// --- Error handling tests ---

func TestCustomErrorReturnsCorrectStatusCode(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/fail", func(r *http.Request, _ struct{}) (*Empty, error) {
		return nil, &NotFoundError{Message: "not found", Detail: "gone"}
	}, shiftapi.WithError[*NotFoundError](http.StatusNotFound))

	resp := doRequest(t, api, http.MethodGet, "/fail", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	body := decodeJSON[map[string]string](t, resp)
	if body["message"] != "not found" {
		t.Errorf("expected message 'not found', got %q", body["message"])
	}
}

func TestCustomErrorReturnsJSON(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/fail", func(r *http.Request, in *Person) (*Greeting, error) {
		return nil, &ConflictError{Code: "CONFLICT", Message: "invalid data"}
	}, shiftapi.WithError[*ConflictError](http.StatusConflict))

	resp := doRequest(t, api, http.MethodPost, "/fail", `{"name":"test"}`)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type application/json; charset=utf-8, got %q", ct)
	}
}

func TestGenericErrorReturns500(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/boom", func(r *http.Request, _ struct{}) (*Empty, error) {
		return nil, errors.New("something broke")
	})

	resp := doRequest(t, api, http.MethodGet, "/boom", "")
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

// --- WithStatus tests ---

func TestWithStatusCustomCode(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/items", func(r *http.Request, in *Item) (*Item, error) {
		in.ID = "new-id"
		return in, nil
	}, shiftapi.WithStatus(http.StatusCreated))

	resp := doRequest(t, api, http.MethodPost, "/items", `{"name":"widget"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestWithStatusOnGetHandler(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Delete(api, "/items/{id}", func(r *http.Request, _ struct{}) (*Empty, error) {
		return &Empty{}, nil
	}, shiftapi.WithStatus(http.StatusNoContent))

	resp := doRequest(t, api, http.MethodDelete, "/items/1", "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

// --- WithRouteInfo tests ---

func TestWithRouteInfoInSpec(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/greet", func(r *http.Request, in *Person) (*Greeting, error) {
		return &Greeting{Hello: in.Name}, nil
	}, shiftapi.WithRouteInfo(shiftapi.RouteInfo{
		Summary:     "Greet someone",
		Description: "Greets a person by name",
		Tags:        []string{"greetings", "social"},
	}))

	spec := api.Spec()
	pathItem := spec.Paths.Find("/greet")
	if pathItem == nil {
		t.Fatal("expected /greet in paths")
	}
	if pathItem.Post == nil {
		t.Fatal("expected POST operation on /greet")
	}
	if pathItem.Post.Summary != "Greet someone" {
		t.Errorf("expected summary %q, got %q", "Greet someone", pathItem.Post.Summary)
	}
	if pathItem.Post.Description != "Greets a person by name" {
		t.Errorf("expected description %q, got %q", "Greets a person by name", pathItem.Post.Description)
	}
	if len(pathItem.Post.Tags) != 2 || pathItem.Post.Tags[0] != "greetings" {
		t.Errorf("expected tags [greetings social], got %v", pathItem.Post.Tags)
	}
}

// --- OpenAPI schema structure tests ---

func TestSpecHasPath(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/health", func(r *http.Request, _ struct{}) (*Status, error) {
		return &Status{OK: true}, nil
	})

	spec := api.Spec()
	if spec.Paths.Find("/health") == nil {
		t.Fatal("expected /health in spec paths")
	}
}

func TestSpecGetHasNoRequestBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/health", func(r *http.Request, _ struct{}) (*Status, error) {
		return &Status{OK: true}, nil
	})

	spec := api.Spec()
	pathItem := spec.Paths.Find("/health")
	if pathItem.Get == nil {
		t.Fatal("expected GET operation")
	}
	if pathItem.Get.RequestBody != nil {
		t.Error("GET should not have a request body in the spec")
	}
}

func TestSpecPostHasRequestBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/greet", func(r *http.Request, in *Person) (*Greeting, error) {
		return &Greeting{Hello: in.Name}, nil
	})

	spec := api.Spec()
	pathItem := spec.Paths.Find("/greet")
	if pathItem.Post == nil {
		t.Fatal("expected POST operation")
	}
	if pathItem.Post.RequestBody == nil {
		t.Error("POST should have a request body in the spec")
	}
}

func TestSpecRequestBodyIsRequired(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/greet", func(r *http.Request, in *Person) (*Greeting, error) {
		return &Greeting{Hello: in.Name}, nil
	})

	spec := api.Spec()
	pathItem := spec.Paths.Find("/greet")
	rb := pathItem.Post.RequestBody
	if rb == nil || rb.Value == nil {
		t.Fatal("expected request body")
	}
	if !rb.Value.Required {
		t.Error("request body should be marked as required")
	}
}

// --- Empty body behavior for body-carrying methods ---

func TestPostNoInputRequiresBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/trigger", func(r *http.Request, _ struct{}) (*Status, error) {
		return &Status{OK: true}, nil
	})

	// Empty body should be rejected
	resp := doRequest(t, api, http.MethodPost, "/trigger", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for POST without body, got %d", resp.StatusCode)
	}

	// Empty JSON object should be accepted
	resp2 := doRequest(t, api, http.MethodPost, "/trigger", `{}`)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for POST with {}, got %d", resp2.StatusCode)
	}
}

func TestPutNoInputRequiresBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Put(api, "/items/{id}", func(r *http.Request, _ struct{}) (*Empty, error) {
		return &Empty{}, nil
	})

	resp := doRequest(t, api, http.MethodPut, "/items/1", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for PUT without body, got %d", resp.StatusCode)
	}

	resp2 := doRequest(t, api, http.MethodPut, "/items/1", `{}`)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for PUT with {}, got %d", resp2.StatusCode)
	}
}

func TestPatchNoInputRequiresBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Patch(api, "/items/{id}", func(r *http.Request, _ struct{}) (*Empty, error) {
		return &Empty{}, nil
	})

	resp := doRequest(t, api, http.MethodPatch, "/items/1", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for PATCH without body, got %d", resp.StatusCode)
	}

	resp2 := doRequest(t, api, http.MethodPatch, "/items/1", `{}`)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for PATCH with {}, got %d", resp2.StatusCode)
	}
}

func TestGetNoInputDoesNotRequireBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/health", func(r *http.Request, _ struct{}) (*Status, error) {
		return &Status{OK: true}, nil
	})

	// GET without body should succeed
	resp := doRequest(t, api, http.MethodGet, "/health", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for GET without body, got %d", resp.StatusCode)
	}
}

func TestDeleteNoInputDoesNotRequireBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Delete(api, "/items/{id}", func(r *http.Request, _ struct{}) (*Empty, error) {
		return &Empty{}, nil
	})

	// DELETE without body should succeed
	resp := doRequest(t, api, http.MethodDelete, "/items/1", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for DELETE without body, got %d", resp.StatusCode)
	}
}

// --- Spec: empty body on body-carrying methods ---

func TestSpecPostNoInputHasEmptyRequestBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/trigger", func(r *http.Request, _ struct{}) (*Status, error) {
		return &Status{OK: true}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/trigger").Post
	if op.RequestBody == nil {
		t.Fatal("POST with no input should still have a request body in the spec")
	}
	if !op.RequestBody.Value.Required {
		t.Error("request body should be required")
	}
	content := op.RequestBody.Value.Content["application/json"]
	if content == nil {
		t.Fatal("expected application/json content")
	}
	if !content.Schema.Value.Type.Is("object") {
		t.Errorf("expected empty object schema, got %v", content.Schema.Value.Type)
	}
	if len(content.Schema.Value.Properties) != 0 {
		t.Errorf("expected 0 properties, got %d", len(content.Schema.Value.Properties))
	}
}

func TestSpecPutNoInputHasEmptyRequestBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Put(api, "/items/{id}", func(r *http.Request, _ struct{}) (*Empty, error) {
		return &Empty{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/items/{id}").Put
	if op.RequestBody == nil {
		t.Fatal("PUT with no input should still have a request body in the spec")
	}
}

func TestSpecGetNoInputHasNoRequestBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/health", func(r *http.Request, _ struct{}) (*Status, error) {
		return &Status{OK: true}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/health").Get
	if op.RequestBody != nil {
		t.Error("GET with no input should not have a request body in the spec")
	}
}

func TestSpecDeleteHasNoRequestBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Delete(api, "/items/{id}", func(r *http.Request, _ struct{}) (*Empty, error) {
		return &Empty{}, nil
	})

	spec := api.Spec()
	pathItem := spec.Paths.Find("/items/{id}")
	if pathItem.Delete == nil {
		t.Fatal("expected DELETE operation")
	}
	if pathItem.Delete.RequestBody != nil {
		t.Error("DELETE should not have a request body in the spec")
	}
}

func TestSpecHasResponseSchema(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/health", func(r *http.Request, _ struct{}) (*Status, error) {
		return &Status{OK: true}, nil
	})

	spec := api.Spec()
	pathItem := spec.Paths.Find("/health")
	resp := pathItem.Get.Responses.Value("200")
	if resp == nil {
		t.Fatal("expected 200 response")
	}
	if resp.Value.Content["application/json"] == nil {
		t.Fatal("expected application/json content in response")
	}
}

func TestSpecResponseDescriptionUsesStatusText(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/health", func(r *http.Request, _ struct{}) (*Status, error) {
		return &Status{OK: true}, nil
	})

	spec := api.Spec()
	resp := spec.Paths.Find("/health").Get.Responses.Value("200")
	if resp.Value.Description == nil || *resp.Value.Description != "OK" {
		t.Errorf("expected response description 'OK', got %v", resp.Value.Description)
	}
}

func TestSpecWithStatusUsesCorrectCodeInSpec(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/items", func(r *http.Request, in *Item) (*Item, error) {
		return in, nil
	}, shiftapi.WithStatus(http.StatusCreated))

	spec := api.Spec()
	pathItem := spec.Paths.Find("/items")
	if pathItem.Post.Responses.Value("201") == nil {
		t.Error("expected 201 response in spec when WithStatus(201) is used")
	}
	if pathItem.Post.Responses.Value("200") != nil {
		t.Error("should not have 200 response when WithStatus(201) is used")
	}
}

func TestSpecComponentSchemasPopulated(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/greet", func(r *http.Request, in *Person) (*Greeting, error) {
		return &Greeting{Hello: in.Name}, nil
	})

	spec := api.Spec()
	if len(spec.Components.Schemas) == 0 {
		t.Fatal("expected component schemas to be populated")
	}
}

func TestSpecMultipleMethodsOnSamePath(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/items", func(r *http.Request, _ struct{}) (*[]Item, error) {
		return &[]Item{}, nil
	})
	shiftapi.Post(api, "/items", func(r *http.Request, in *Item) (*Item, error) {
		return in, nil
	})

	spec := api.Spec()
	pathItem := spec.Paths.Find("/items")
	if pathItem == nil {
		t.Fatal("expected /items in paths")
	}
	if pathItem.Get == nil {
		t.Error("expected GET on /items")
	}
	if pathItem.Post == nil {
		t.Error("expected POST on /items")
	}
}

func TestSpecOpenAPIVersion(t *testing.T) {
	api := newTestAPI(t)
	if api.Spec().OpenAPI != "3.1" {
		t.Errorf("expected OpenAPI 3.1, got %q", api.Spec().OpenAPI)
	}
}

// --- Path parameter spec tests ---

func TestSpecPathParametersDocumented(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/users/{id}", func(r *http.Request, _ struct{}) (*Item, error) {
		return &Item{ID: r.PathValue("id")}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/users/{id}").Get
	if len(op.Parameters) != 1 {
		t.Fatalf("expected 1 path parameter, got %d", len(op.Parameters))
	}
	param := op.Parameters[0].Value
	if param.Name != "id" {
		t.Errorf("expected parameter name 'id', got %q", param.Name)
	}
	if param.In != "path" {
		t.Errorf("expected parameter in 'path', got %q", param.In)
	}
	if !param.Required {
		t.Error("path parameters must be required")
	}
}

func TestSpecMultiplePathParameters(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/orgs/{orgId}/users/{userId}", func(r *http.Request, _ struct{}) (*Item, error) {
		return &Item{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/orgs/{orgId}/users/{userId}").Get
	if len(op.Parameters) != 2 {
		t.Fatalf("expected 2 path parameters, got %d", len(op.Parameters))
	}
	if op.Parameters[0].Value.Name != "orgId" {
		t.Errorf("expected first param 'orgId', got %q", op.Parameters[0].Value.Name)
	}
	if op.Parameters[1].Value.Name != "userId" {
		t.Errorf("expected second param 'userId', got %q", op.Parameters[1].Value.Name)
	}
}

func TestSpecNoPathParametersWhenNoneInPath(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/health", func(r *http.Request, _ struct{}) (*Status, error) {
		return &Status{OK: true}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/health").Get
	if len(op.Parameters) != 0 {
		t.Errorf("expected 0 parameters, got %d", len(op.Parameters))
	}
}

// --- Operation ID tests ---

func TestSpecOperationID(t *testing.T) {
	tests := []struct {
		method     string
		path       string
		expectedID string
	}{
		{"GET", "/health", "getHealth"},
		{"GET", "/users/{id}", "getUsersById"},
		{"POST", "/users", "postUsers"},
		{"DELETE", "/orgs/{orgId}/users/{userId}", "deleteOrgsByOrgIdUsersByUserId"},
		{"PUT", "/items/{id}", "putItemsById"},
	}

	for _, tc := range tests {
		t.Run(tc.expectedID, func(t *testing.T) {
			api := newTestAPI(t)
			switch tc.method {
			case "GET":
				shiftapi.Get(api, tc.path, func(r *http.Request, _ struct{}) (*Empty, error) {
					return &Empty{}, nil
				})
			case "POST":
				shiftapi.Post(api, tc.path, func(r *http.Request, in *Empty) (*Empty, error) {
					return &Empty{}, nil
				})
			case "PUT":
				shiftapi.Put(api, tc.path, func(r *http.Request, in *Empty) (*Empty, error) {
					return &Empty{}, nil
				})
			case "DELETE":
				shiftapi.Delete(api, tc.path, func(r *http.Request, _ struct{}) (*Empty, error) {
					return &Empty{}, nil
				})
			}

			spec := api.Spec()
			pathItem := spec.Paths.Find(tc.path)
			var op *openapi3.Operation
			switch tc.method {
			case "GET":
				op = pathItem.Get
			case "POST":
				op = pathItem.Post
			case "PUT":
				op = pathItem.Put
			case "DELETE":
				op = pathItem.Delete
			}
			if op.OperationID != tc.expectedID {
				t.Errorf("expected operationId %q, got %q", tc.expectedID, op.OperationID)
			}
		})
	}
}

// --- Default error response tests ---

func TestSpecHas422And500ErrorResponses(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/health", func(r *http.Request, _ struct{}) (*Status, error) {
		return &Status{OK: true}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/health").Get

	// 422 ValidationError
	resp422 := op.Responses.Value("422")
	if resp422 == nil {
		t.Fatal("expected 422 error response in spec")
	}
	if resp422.Value.Description == nil || *resp422.Value.Description != "Validation Error" {
		t.Error("expected 422 response description 'Validation Error'")
	}
	content422 := resp422.Value.Content["application/json"]
	if content422 == nil {
		t.Fatal("expected application/json content in 422 response")
	}
	if content422.Schema.Ref != "#/components/schemas/ValidationError" {
		t.Errorf("expected 422 schema ref to ValidationError, got %s", content422.Schema.Ref)
	}

	// 500 APIError
	resp500 := op.Responses.Value("500")
	if resp500 == nil {
		t.Fatal("expected 500 error response in spec")
	}
	if resp500.Value.Description == nil || *resp500.Value.Description != "Internal Server Error" {
		t.Error("expected 500 response description 'Internal Server Error'")
	}
	content500 := resp500.Value.Content["application/json"]
	if content500 == nil {
		t.Fatal("expected application/json content in 500 response")
	}
	if content500.Schema.Ref != "#/components/schemas/InternalServerError" {
		t.Errorf("expected 500 schema ref to APIError, got %s", content500.Schema.Ref)
	}
}

func TestSpecErrorResponsesOnPost(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/items", func(r *http.Request, in *Item) (*Item, error) {
		return in, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/items").Post
	if op.Responses.Value("422") == nil {
		t.Fatal("expected 422 error response on POST")
	}
	if op.Responses.Value("500") == nil {
		t.Fatal("expected 500 error response on POST")
	}
}

// --- HTTP Handler interface tests ---

func TestAPIImplementsHTTPHandler(t *testing.T) {
	var _ http.Handler = shiftapi.New()
}

func TestHTTPTestServerCompatibility(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/ping", func(r *http.Request, _ struct{}) (*Status, error) {
		return &Status{OK: true}, nil
	})

	srv := httptest.NewServer(api)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/ping")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	status := decodeJSON[Status](t, resp)
	if !status.OK {
		t.Error("expected OK=true")
	}
}

// --- Middleware compatibility tests ---

func TestMiddlewareCompatibility(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/health", func(r *http.Request, _ struct{}) (*Status, error) {
		return &Status{OK: true}, nil
	})

	// Wrap with a simple header-adding middleware
	wrapped := addHeaderMiddleware("X-Custom", "test-value")(api)

	resp := doRequest(t, wrapped, http.MethodGet, "/health", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if v := resp.Header.Get("X-Custom"); v != "test-value" {
		t.Errorf("expected X-Custom=test-value, got %q", v)
	}
}

func addHeaderMiddleware(key, value string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(key, value)
			next.ServeHTTP(w, r)
		})
	}
}

// --- Mux composition tests ---

func TestMountUnderPrefix(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/health", func(r *http.Request, _ struct{}) (*Status, error) {
		return &Status{OK: true}, nil
	})

	mux := http.NewServeMux()
	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", api))

	resp := doRequest(t, mux, http.MethodGet, "/api/v1/health", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	status := decodeJSON[Status](t, resp)
	if !status.OK {
		t.Error("expected OK=true from mounted API")
	}
}

// --- Request access tests ---

func TestHandlerAccessesHeaders(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/echo-header", func(r *http.Request, _ struct{}) (*map[string]string, error) {
		return &map[string]string{
			"value": r.Header.Get("X-Test"),
		}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/echo-header", nil)
	req.Header.Set("X-Test", "hello")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	resp := rec.Result()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[map[string]string](t, resp)
	if result["value"] != "hello" {
		t.Errorf("expected header value 'hello', got %q", result["value"])
	}
}

func TestHandlerAccessesQueryParams(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/search", func(r *http.Request, _ struct{}) (*map[string]string, error) {
		return &map[string]string{
			"q": r.URL.Query().Get("q"),
		}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/search?q=hello", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[map[string]string](t, resp)
	if result["q"] != "hello" {
		t.Errorf("expected q=hello, got %q", result["q"])
	}
}

func TestHandlerAccessesContext(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/ctx", func(r *http.Request, _ struct{}) (*Status, error) {
		if r.Context() == nil {
			return nil, errors.New("context is nil")
		}
		return &Status{OK: true}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/ctx", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// --- Response Content-Type tests ---

func TestSuccessResponseHasJSONContentType(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/test", func(r *http.Request, _ struct{}) (*Status, error) {
		return &Status{OK: true}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/test", "")
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type application/json; charset=utf-8, got %q", ct)
	}
}

func TestErrorResponseHasJSONContentType(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/fail", func(r *http.Request, _ struct{}) (*Empty, error) {
		return nil, &ConflictError{Code: "BAD", Message: "bad"}
	}, shiftapi.WithError[*ConflictError](http.StatusConflict))

	resp := doRequest(t, api, http.MethodGet, "/fail", "")
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type application/json; charset=utf-8, got %q", ct)
	}
}

// --- Multiple routes tests ---

func TestMultipleRoutes(t *testing.T) {
	api := newTestAPI(t)

	shiftapi.Get(api, "/a", func(r *http.Request, _ struct{}) (*map[string]string, error) {
		return &map[string]string{"route": "a"}, nil
	})
	shiftapi.Get(api, "/b", func(r *http.Request, _ struct{}) (*map[string]string, error) {
		return &map[string]string{"route": "b"}, nil
	})
	shiftapi.Post(api, "/c", func(r *http.Request, in *Empty) (*map[string]string, error) {
		return &map[string]string{"route": "c"}, nil
	})

	for _, tc := range []struct {
		method string
		path   string
		body   string
		route  string
	}{
		{http.MethodGet, "/a", "", "a"},
		{http.MethodGet, "/b", "", "b"},
		{http.MethodPost, "/c", `{}`, "c"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			resp := doRequest(t, api, tc.method, tc.path, tc.body)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200, got %d", resp.StatusCode)
			}
			result := decodeJSON[map[string]string](t, resp)
			if result["route"] != tc.route {
				t.Errorf("expected route=%s, got %q", tc.route, result["route"])
			}
		})
	}
}

// --- Spec() method tests ---

func TestSpecReturnsNonNil(t *testing.T) {
	api := shiftapi.New()
	if api.Spec() == nil {
		t.Fatal("Spec() should not return nil")
	}
}

func TestSpecReturnsLiveObject(t *testing.T) {
	api := newTestAPI(t)
	before := len(api.Spec().Paths.InMatchingOrder())

	shiftapi.Get(api, "/new-route", func(r *http.Request, _ struct{}) (*Empty, error) {
		return &Empty{}, nil
	})

	after := len(api.Spec().Paths.InMatchingOrder())
	if after <= before {
		t.Error("expected Spec() to reflect newly registered routes")
	}
}

// --- Validation test types ---

type ValidatedPerson struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

type MinMaxBody struct {
	Age  int    `json:"age" validate:"min=1,max=150"`
	Name string `json:"name" validate:"min=2,max=50"`
}

type OneOfBody struct {
	Status string `json:"status" validate:"oneof=active inactive pending"`
}

type NoValidateBody struct {
	Foo string `json:"foo"`
}

// --- Validation runtime tests ---

func TestValidationRequiredFieldMissing(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/person", func(r *http.Request, in *ValidatedPerson) (*ValidatedPerson, error) {
		return in, nil
	})

	resp := doRequest(t, api, http.MethodPost, "/person", `{"email":"test@example.com"}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	_ = resp.Body.Close()

	if result["message"] != "validation failed" {
		t.Errorf("expected message 'validation failed', got %v", result["message"])
	}
	errs, ok := result["errors"].([]any)
	if !ok || len(errs) == 0 {
		t.Fatal("expected errors array")
	}
	firstErr := errs[0].(map[string]any)
	if firstErr["field"] != "Name" {
		t.Errorf("expected field 'Name', got %v", firstErr["field"])
	}
	if firstErr["message"] != "this field is required" {
		t.Errorf("expected message 'this field is required', got %v", firstErr["message"])
	}
}

func TestValidationEmailInvalid(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/person", func(r *http.Request, in *ValidatedPerson) (*ValidatedPerson, error) {
		return in, nil
	})

	resp := doRequest(t, api, http.MethodPost, "/person", `{"name":"alice","email":"not-an-email"}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	_ = resp.Body.Close()

	errs := result["errors"].([]any)
	found := false
	for _, e := range errs {
		fe := e.(map[string]any)
		if fe["field"] == "Email" && fe["message"] == "must be a valid email address" {
			found = true
		}
	}
	if !found {
		t.Error("expected email validation error")
	}
}

func TestValidationMinMaxViolated(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/minmax", func(r *http.Request, in *MinMaxBody) (*MinMaxBody, error) {
		return in, nil
	})

	resp := doRequest(t, api, http.MethodPost, "/minmax", `{"age":0,"name":"a"}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestValidationValidPayloadPassesThrough(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/person", func(r *http.Request, in *ValidatedPerson) (*ValidatedPerson, error) {
		return in, nil
	})

	resp := doRequest(t, api, http.MethodPost, "/person", `{"name":"alice","email":"alice@example.com"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[ValidatedPerson](t, resp)
	if result.Name != "alice" {
		t.Errorf("expected Name=alice, got %q", result.Name)
	}
}

func TestValidationNoTagsPassThrough(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/noval", func(r *http.Request, in *NoValidateBody) (*NoValidateBody, error) {
		return in, nil
	})

	resp := doRequest(t, api, http.MethodPost, "/noval", `{"foo":"bar"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestWithValidatorCustomInstance(t *testing.T) {
	v := validator.New()
	api := shiftapi.New(shiftapi.WithValidator(v))
	shiftapi.Post(api, "/person", func(r *http.Request, in *ValidatedPerson) (*ValidatedPerson, error) {
		return in, nil
	})

	// Missing required fields should still fail
	resp := doRequest(t, api, http.MethodPost, "/person", `{}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestValidationErrorSatisfiesErrorsAs(t *testing.T) {
	err := &shiftapi.ValidationError{
		Message: "validation failed",
		Errors:  []shiftapi.FieldError{{Field: "Name", Message: "required"}},
	}
	valErr, ok := errors.AsType[*shiftapi.ValidationError](err)
	if !ok {
		t.Fatal("expected errors.AsType to match *ValidationError")
	}
	if valErr.Message != "validation failed" {
		t.Errorf("expected message 'validation failed', got %q", valErr.Message)
	}
}

// --- Validation spec tests ---

func TestSpecRequiredFieldInParentSchema(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/person", func(r *http.Request, in *ValidatedPerson) (*ValidatedPerson, error) {
		return in, nil
	})

	spec := api.Spec()
	// Find the ValidatedPerson schema in components
	schemaRef, ok := spec.Components.Schemas["ValidatedPerson"]
	if !ok {
		t.Fatal("expected ValidatedPerson in component schemas")
	}
	schema := schemaRef.Value
	if !slices.Contains(schema.Required, "name") {
		t.Errorf("expected 'name' in required, got %v", schema.Required)
	}
	if !slices.Contains(schema.Required, "email") {
		t.Errorf("expected 'email' in required, got %v", schema.Required)
	}
}

func TestSpecEmailFormatSet(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/person", func(r *http.Request, in *ValidatedPerson) (*ValidatedPerson, error) {
		return in, nil
	})

	spec := api.Spec()
	schemaRef := spec.Components.Schemas["ValidatedPerson"]
	emailProp := schemaRef.Value.Properties["email"]
	if emailProp == nil {
		t.Fatal("expected 'email' property")
	}
	if emailProp.Value.Format != "email" {
		t.Errorf("expected format 'email', got %q", emailProp.Value.Format)
	}
}

func TestSpecMinMaxOnNumber(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/minmax", func(r *http.Request, in *MinMaxBody) (*MinMaxBody, error) {
		return in, nil
	})

	spec := api.Spec()
	schemaRef := spec.Components.Schemas["MinMaxBody"]
	ageProp := schemaRef.Value.Properties["age"]
	if ageProp == nil {
		t.Fatal("expected 'age' property")
	}
	if ageProp.Value.Min == nil || *ageProp.Value.Min != 1 {
		t.Errorf("expected minimum 1, got %v", ageProp.Value.Min)
	}
	if ageProp.Value.Max == nil || *ageProp.Value.Max != 150 {
		t.Errorf("expected maximum 150, got %v", ageProp.Value.Max)
	}
}

func TestSpecMinMaxOnString(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/minmax", func(r *http.Request, in *MinMaxBody) (*MinMaxBody, error) {
		return in, nil
	})

	spec := api.Spec()
	schemaRef := spec.Components.Schemas["MinMaxBody"]
	nameProp := schemaRef.Value.Properties["name"]
	if nameProp == nil {
		t.Fatal("expected 'name' property")
	}
	if nameProp.Value.MinLength != 2 {
		t.Errorf("expected minLength 2, got %d", nameProp.Value.MinLength)
	}
	if nameProp.Value.MaxLength == nil || *nameProp.Value.MaxLength != 50 {
		t.Errorf("expected maxLength 50, got %v", nameProp.Value.MaxLength)
	}
}

func TestSpecEnumOnOneOf(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/oneof", func(r *http.Request, in *OneOfBody) (*OneOfBody, error) {
		return in, nil
	})

	spec := api.Spec()
	schemaRef := spec.Components.Schemas["OneOfBody"]
	statusProp := schemaRef.Value.Properties["status"]
	if statusProp == nil {
		t.Fatal("expected 'status' property")
	}
	if len(statusProp.Value.Enum) != 3 {
		t.Fatalf("expected 3 enum values, got %d", len(statusProp.Value.Enum))
	}
	expected := []string{"active", "inactive", "pending"}
	for i, v := range statusProp.Value.Enum {
		if v != expected[i] {
			t.Errorf("expected enum[%d]=%q, got %v", i, expected[i], v)
		}
	}
}

// --- Nested struct validation tests ---

type Address struct {
	Street string `json:"street" validate:"required"`
	City   string `json:"city" validate:"required"`
}

type PersonWithAddress struct {
	Name    string  `json:"name" validate:"required"`
	Address Address `json:"address" validate:"required"`
}

func TestValidationNestedStructValid(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/person-addr", func(r *http.Request, in *PersonWithAddress) (*PersonWithAddress, error) {
		return in, nil
	})

	resp := doRequest(t, api, http.MethodPost, "/person-addr", `{"name":"alice","address":{"street":"123 Main St","city":"Springfield"}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestValidationNestedStructMissingFields(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/person-addr", func(r *http.Request, in *PersonWithAddress) (*PersonWithAddress, error) {
		return in, nil
	})

	resp := doRequest(t, api, http.MethodPost, "/person-addr", `{"name":"alice","address":{}}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

// --- Query parameter test types ---

type SearchQuery struct {
	Q     string `query:"q" validate:"required"`
	Page  int    `query:"page" validate:"min=1"`
	Limit int    `query:"limit" validate:"min=1,max=100"`
}

type SearchResult struct {
	Query string `json:"query"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}

type TagQuery struct {
	Tags []string `query:"tag"`
}

type TagResult struct {
	Tags []string `json:"tags"`
}

type OptionalQuery struct {
	Name  string `query:"name"`
	Debug *bool  `query:"debug"`
	Limit *int   `query:"limit"`
}

type OptionalResult struct {
	Name     string `json:"name"`
	HasDebug bool   `json:"has_debug"`
	Debug    bool   `json:"debug"`
	HasLimit bool   `json:"has_limit"`
	Limit    int    `json:"limit"`
}

type FilterQuery struct {
	Status string `query:"status" validate:"oneof=active inactive pending"`
}

// --- Query parameter runtime tests ---

func TestGetWithQueryBasic(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/search", func(r *http.Request, in SearchQuery) (*SearchResult, error) {
		return &SearchResult{Query: in.Q, Page: in.Page, Limit: in.Limit}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/search?q=hello&page=2&limit=10", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[SearchResult](t, resp)
	if result.Query != "hello" {
		t.Errorf("expected Query=hello, got %q", result.Query)
	}
	if result.Page != 2 {
		t.Errorf("expected Page=2, got %d", result.Page)
	}
	if result.Limit != 10 {
		t.Errorf("expected Limit=10, got %d", result.Limit)
	}
}

func TestGetWithQueryMissingRequired(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/search", func(r *http.Request, in SearchQuery) (*SearchResult, error) {
		return &SearchResult{}, nil
	})

	// Missing required "q" param
	resp := doRequest(t, api, http.MethodGet, "/search?page=1&limit=10", "")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestGetWithQueryInvalidType(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/search", func(r *http.Request, in SearchQuery) (*SearchResult, error) {
		return &SearchResult{}, nil
	})

	// "page" should be an int, not "abc"
	resp := doRequest(t, api, http.MethodGet, "/search?q=test&page=abc&limit=10", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetWithQuerySliceParams(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/tags", func(r *http.Request, in TagQuery) (*TagResult, error) {
		return &TagResult{Tags: in.Tags}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/tags?tag=a&tag=b&tag=c", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[TagResult](t, resp)
	if len(result.Tags) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(result.Tags))
	}
	expected := []string{"a", "b", "c"}
	for i, tag := range result.Tags {
		if tag != expected[i] {
			t.Errorf("expected tag[%d]=%q, got %q", i, expected[i], tag)
		}
	}
}

func TestGetWithQueryOptionalPointer(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/optional", func(r *http.Request, in OptionalQuery) (*OptionalResult, error) {
		result := &OptionalResult{Name: in.Name}
		if in.Debug != nil {
			result.HasDebug = true
			result.Debug = *in.Debug
		}
		if in.Limit != nil {
			result.HasLimit = true
			result.Limit = *in.Limit
		}
		return result, nil
	})

	// With optional params
	resp := doRequest(t, api, http.MethodGet, "/optional?name=test&debug=true&limit=50", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[OptionalResult](t, resp)
	if !result.HasDebug || !result.Debug {
		t.Error("expected debug=true")
	}
	if !result.HasLimit || result.Limit != 50 {
		t.Error("expected limit=50")
	}

	// Without optional params
	resp2 := doRequest(t, api, http.MethodGet, "/optional?name=test", "")
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}
	result2 := decodeJSON[OptionalResult](t, resp2)
	if result2.HasDebug {
		t.Error("expected debug to be absent")
	}
	if result2.HasLimit {
		t.Error("expected limit to be absent")
	}
}

func TestPostWithQueryAndBody(t *testing.T) {
	api := newTestAPI(t)

	type CreateInput struct {
		DryRun bool   `query:"dry_run"`
		Name   string `json:"name"`
	}
	type CreateResult struct {
		Name   string `json:"name"`
		DryRun bool   `json:"dry_run"`
	}

	shiftapi.Post(api, "/items", func(r *http.Request, in CreateInput) (*CreateResult, error) {
		return &CreateResult{Name: in.Name, DryRun: in.DryRun}, nil
	})

	resp := doRequest(t, api, http.MethodPost, "/items?dry_run=true", `{"name":"widget"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[CreateResult](t, resp)
	if result.Name != "widget" {
		t.Errorf("expected Name=widget, got %q", result.Name)
	}
	if !result.DryRun {
		t.Error("expected DryRun=true")
	}
}

// --- Query/JSON tag interop tests ---

func TestQueryFieldInBodyIsIgnored(t *testing.T) {
	api := newTestAPI(t)

	type Input struct {
		DryRun bool   `query:"dry_run"`
		Name   string `json:"name"`
	}

	shiftapi.Post(api, "/items", func(r *http.Request, in Input) (*map[string]any, error) {
		return &map[string]any{"name": in.Name, "dry_run": in.DryRun}, nil
	})

	// Use the Go field name "DryRun" which json.Decode would match
	// case-insensitively — resetQueryFields must clear it.
	resp := doRequest(t, api, http.MethodPost, "/items", `{"name":"widget","DryRun":true}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[map[string]any](t, resp)
	if result["name"] != "widget" {
		t.Errorf("expected name=widget, got %v", result["name"])
	}
	if result["dry_run"] != false {
		t.Errorf("expected dry_run=false (query field must not be set from body), got %v", result["dry_run"])
	}
}

func TestBodyFieldInQueryIsIgnored(t *testing.T) {
	api := newTestAPI(t)

	type Input struct {
		DryRun bool   `query:"dry_run"`
		Name   string `json:"name"`
	}

	shiftapi.Post(api, "/items", func(r *http.Request, in Input) (*map[string]any, error) {
		return &map[string]any{"name": in.Name, "dry_run": in.DryRun}, nil
	})

	// Send name in query but NOT in body — should remain empty
	resp := doRequest(t, api, http.MethodPost, "/items?name=sneaky&dry_run=true", `{"name":"widget"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[map[string]any](t, resp)
	if result["name"] != "widget" {
		t.Errorf("expected name from body (widget), got %v", result["name"])
	}
	if result["dry_run"] != true {
		t.Errorf("expected dry_run=true from query, got %v", result["dry_run"])
	}
}

func TestFieldWithBothJsonAndQueryTagsUsesQuery(t *testing.T) {
	api := newTestAPI(t)

	type Input struct {
		Mode string `query:"mode" json:"mode"`
		Name string `json:"name"`
	}

	shiftapi.Post(api, "/items", func(r *http.Request, in Input) (*map[string]string, error) {
		return &map[string]string{"mode": in.Mode, "name": in.Name}, nil
	})

	// Send conflicting values: "body_mode" in body, "query_mode" in query
	resp := doRequest(t, api, http.MethodPost, "/items?mode=query_mode", `{"name":"widget","mode":"body_mode"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[map[string]string](t, resp)
	// Query parsing runs after body decode, so query value should win
	if result["mode"] != "query_mode" {
		t.Errorf("expected mode=query_mode (query overrides body), got %q", result["mode"])
	}
	if result["name"] != "widget" {
		t.Errorf("expected name=widget, got %q", result["name"])
	}
}

func TestSpecMixedStructBodyExcludesQueryFields(t *testing.T) {
	api := newTestAPI(t)

	type Input struct {
		DryRun bool   `query:"dry_run"`
		Name   string `json:"name"`
		ID     string `json:"id"`
	}

	shiftapi.Post(api, "/items", func(r *http.Request, in Input) (*Empty, error) {
		return &Empty{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/items").Post

	// Query params should include dry_run only
	queryParams := 0
	for _, p := range op.Parameters {
		if p.Value.In == "query" {
			queryParams++
			if p.Value.Name != "dry_run" {
				t.Errorf("unexpected query param %q", p.Value.Name)
			}
		}
	}
	if queryParams != 1 {
		t.Errorf("expected 1 query parameter, got %d", queryParams)
	}

	// Body schema should include name and id but NOT dry_run
	if op.RequestBody == nil {
		t.Fatal("expected request body")
	}
	bodyRef := op.RequestBody.Value.Content["application/json"].Schema.Ref
	schemaName := bodyRef[len("#/components/schemas/"):]
	bodySchema := spec.Components.Schemas[schemaName].Value
	if bodySchema.Properties["name"] == nil {
		t.Error("expected 'name' in body schema")
	}
	if bodySchema.Properties["id"] == nil {
		t.Error("expected 'id' in body schema")
	}
	if bodySchema.Properties["dry_run"] != nil {
		t.Error("'dry_run' should NOT be in body schema (it's a query param)")
	}
}

func TestSpecMixedStructQueryExcludesBodyFields(t *testing.T) {
	api := newTestAPI(t)

	type Input struct {
		Filter string `query:"filter"`
		Sort   string `query:"sort"`
		Name   string `json:"name"`
	}

	shiftapi.Post(api, "/items", func(r *http.Request, in Input) (*Empty, error) {
		return &Empty{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/items").Post

	// Query params should only be filter and sort, not name
	paramNames := map[string]bool{}
	for _, p := range op.Parameters {
		if p.Value.In == "query" {
			paramNames[p.Value.Name] = true
		}
	}
	if !paramNames["filter"] {
		t.Error("expected 'filter' query param")
	}
	if !paramNames["sort"] {
		t.Error("expected 'sort' query param")
	}
	if paramNames["name"] {
		t.Error("'name' should NOT be a query param (it's a body field)")
	}
}

func TestGetWithQueryAndPathParams(t *testing.T) {
	api := newTestAPI(t)

	type ItemQuery struct {
		Fields string `query:"fields"`
	}

	shiftapi.Get(api, "/items/{id}", func(r *http.Request, in ItemQuery) (*map[string]string, error) {
		return &map[string]string{
			"id":     r.PathValue("id"),
			"fields": in.Fields,
		}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/items/abc123?fields=name,price", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[map[string]string](t, resp)
	if result["id"] != "abc123" {
		t.Errorf("expected id=abc123, got %q", result["id"])
	}
	if result["fields"] != "name,price" {
		t.Errorf("expected fields=name,price, got %q", result["fields"])
	}
}

func TestDeleteWithQuery(t *testing.T) {
	api := newTestAPI(t)

	type DeleteQuery struct {
		Force bool `query:"force"`
	}

	shiftapi.Delete(api, "/items/{id}", func(r *http.Request, in DeleteQuery) (*map[string]any, error) {
		return &map[string]any{
			"id":    r.PathValue("id"),
			"force": in.Force,
		}, nil
	})

	resp := doRequest(t, api, http.MethodDelete, "/items/42?force=true", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetWithQueryValidationConstraint(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/filter", func(r *http.Request, in FilterQuery) (*map[string]string, error) {
		return &map[string]string{"status": in.Status}, nil
	})

	// Valid value
	resp := doRequest(t, api, http.MethodGet, "/filter?status=active", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Invalid value -> 422
	resp2 := doRequest(t, api, http.MethodGet, "/filter?status=unknown", "")
	if resp2.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp2.StatusCode)
	}
}

// --- Query parameter: scalar type parsing ---

func TestGetWithQueryBoolScalar(t *testing.T) {
	api := newTestAPI(t)

	type BoolQuery struct {
		Verbose bool `query:"verbose"`
	}

	shiftapi.Get(api, "/logs", func(r *http.Request, in BoolQuery) (*map[string]bool, error) {
		return &map[string]bool{"verbose": in.Verbose}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/logs?verbose=true", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[map[string]bool](t, resp)
	if !result["verbose"] {
		t.Error("expected verbose=true")
	}

	// false value
	resp2 := doRequest(t, api, http.MethodGet, "/logs?verbose=false", "")
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}
	result2 := decodeJSON[map[string]bool](t, resp2)
	if result2["verbose"] {
		t.Error("expected verbose=false")
	}
}

func TestGetWithQueryUint(t *testing.T) {
	api := newTestAPI(t)

	type PageQuery struct {
		Offset uint   `query:"offset"`
		Limit  uint64 `query:"limit"`
	}

	shiftapi.Get(api, "/pages", func(r *http.Request, in PageQuery) (*map[string]uint64, error) {
		return &map[string]uint64{"offset": uint64(in.Offset), "limit": in.Limit}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/pages?offset=10&limit=100", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[map[string]float64](t, resp)
	if result["offset"] != 10 {
		t.Errorf("expected offset=10, got %v", result["offset"])
	}
	if result["limit"] != 100 {
		t.Errorf("expected limit=100, got %v", result["limit"])
	}
}

func TestGetWithQueryFloat(t *testing.T) {
	api := newTestAPI(t)

	type CoordQuery struct {
		Lat float64 `query:"lat"`
		Lng float32 `query:"lng"`
	}

	shiftapi.Get(api, "/nearby", func(r *http.Request, in CoordQuery) (*map[string]float64, error) {
		return &map[string]float64{"lat": in.Lat, "lng": float64(in.Lng)}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/nearby?lat=40.7128&lng=-74.006", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[map[string]float64](t, resp)
	if result["lat"] != 40.7128 {
		t.Errorf("expected lat=40.7128, got %v", result["lat"])
	}
}

// --- Query parameter: parse errors ---

func TestGetWithQueryInvalidBool(t *testing.T) {
	api := newTestAPI(t)

	type BoolQuery struct {
		Debug bool `query:"debug"`
	}

	shiftapi.Get(api, "/test", func(r *http.Request, in BoolQuery) (*Empty, error) {
		return &Empty{}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/test?debug=notabool", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetWithQueryInvalidUint(t *testing.T) {
	api := newTestAPI(t)

	type UintQuery struct {
		Count uint `query:"count"`
	}

	shiftapi.Get(api, "/test", func(r *http.Request, in UintQuery) (*Empty, error) {
		return &Empty{}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/test?count=-1", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetWithQueryInvalidFloat(t *testing.T) {
	api := newTestAPI(t)

	type FloatQuery struct {
		Score float64 `query:"score"`
	}

	shiftapi.Get(api, "/test", func(r *http.Request, in FloatQuery) (*Empty, error) {
		return &Empty{}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/test?score=abc", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// --- Query parameter: skip and zero-value behavior ---

func TestGetWithQuerySkipTag(t *testing.T) {
	api := newTestAPI(t)

	type SkipQuery struct {
		Name   string `query:"name"`
		Secret string `json:"-"`
	}

	shiftapi.Get(api, "/test", func(r *http.Request, in SkipQuery) (*map[string]string, error) {
		return &map[string]string{"name": in.Name, "secret": in.Secret}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/test?name=alice&secret=hidden", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[map[string]string](t, resp)
	if result["name"] != "alice" {
		t.Errorf("expected name=alice, got %q", result["name"])
	}
	if result["secret"] != "" {
		t.Errorf("expected secret to be empty (skipped), got %q", result["secret"])
	}
}

func TestGetWithQueryAbsentParamsGetZeroValues(t *testing.T) {
	api := newTestAPI(t)

	type MixedQuery struct {
		Name  string `query:"name"`
		Count int    `query:"count"`
		Flag  bool   `query:"flag"`
	}

	shiftapi.Get(api, "/test", func(r *http.Request, in MixedQuery) (*map[string]any, error) {
		return &map[string]any{
			"name":  in.Name,
			"count": in.Count,
			"flag":  in.Flag,
		}, nil
	})

	// No query params at all — everything should be zero-valued
	resp := doRequest(t, api, http.MethodGet, "/test", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[map[string]any](t, resp)
	if result["name"] != "" {
		t.Errorf("expected name=\"\", got %v", result["name"])
	}
	if result["count"] != float64(0) {
		t.Errorf("expected count=0, got %v", result["count"])
	}
	if result["flag"] != false {
		t.Errorf("expected flag=false, got %v", result["flag"])
	}
}

// --- Query parameter: spec types for bool/float/uint ---

func TestSpecQueryParamBoolType(t *testing.T) {
	api := newTestAPI(t)

	type BoolQuery struct {
		Debug bool `query:"debug"`
	}

	shiftapi.Get(api, "/test", func(r *http.Request, in BoolQuery) (*Empty, error) {
		return &Empty{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/test").Get
	if len(op.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(op.Parameters))
	}
	if !op.Parameters[0].Value.Schema.Value.Type.Is("boolean") {
		t.Errorf("expected boolean type, got %v", op.Parameters[0].Value.Schema.Value.Type)
	}
}

func TestSpecQueryParamFloatType(t *testing.T) {
	api := newTestAPI(t)

	type FloatQuery struct {
		Score float64 `query:"score"`
	}

	shiftapi.Get(api, "/test", func(r *http.Request, in FloatQuery) (*Empty, error) {
		return &Empty{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/test").Get
	if len(op.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(op.Parameters))
	}
	if !op.Parameters[0].Value.Schema.Value.Type.Is("number") {
		t.Errorf("expected number type, got %v", op.Parameters[0].Value.Schema.Value.Type)
	}
}

func TestSpecQueryParamUintType(t *testing.T) {
	api := newTestAPI(t)

	type UintQuery struct {
		Count uint `query:"count"`
	}

	shiftapi.Get(api, "/test", func(r *http.Request, in UintQuery) (*Empty, error) {
		return &Empty{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/test").Get
	if len(op.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(op.Parameters))
	}
	if !op.Parameters[0].Value.Schema.Value.Type.Is("integer") {
		t.Errorf("expected integer type, got %v", op.Parameters[0].Value.Schema.Value.Type)
	}
}

func TestSpecQuerySkipTagNotDocumented(t *testing.T) {
	api := newTestAPI(t)

	type SkipQuery struct {
		Name   string `query:"name"`
		Secret string `json:"-"`
	}

	shiftapi.Get(api, "/test", func(r *http.Request, in SkipQuery) (*Empty, error) {
		return &Empty{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/test").Get
	if len(op.Parameters) != 1 {
		t.Fatalf("expected 1 parameter (secret should be skipped), got %d", len(op.Parameters))
	}
	if op.Parameters[0].Value.Name != "name" {
		t.Errorf("expected parameter 'name', got %q", op.Parameters[0].Value.Name)
	}
}

func TestSpecQueryParamOptionalPointerNotRequired(t *testing.T) {
	api := newTestAPI(t)

	shiftapi.Get(api, "/optional", func(r *http.Request, in OptionalQuery) (*Empty, error) {
		return &Empty{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/optional").Get

	for _, p := range op.Parameters {
		if p.Value.Required {
			t.Errorf("expected parameter %q to not be required (pointer type)", p.Value.Name)
		}
	}
}

// --- Query parameter spec tests ---

func TestSpecQueryParamsDocumented(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/search", func(r *http.Request, in SearchQuery) (*SearchResult, error) {
		return &SearchResult{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/search").Get
	if op == nil {
		t.Fatal("expected GET operation on /search")
	}

	// Should have 3 query params: q, page, limit
	queryParams := 0
	for _, p := range op.Parameters {
		if p.Value.In == "query" {
			queryParams++
		}
	}
	if queryParams != 3 {
		t.Fatalf("expected 3 query parameters, got %d", queryParams)
	}
}

func TestSpecQueryParamTypes(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/search", func(r *http.Request, in SearchQuery) (*SearchResult, error) {
		return &SearchResult{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/search").Get

	paramByName := make(map[string]*openapi3.Parameter)
	for _, p := range op.Parameters {
		paramByName[p.Value.Name] = p.Value
	}

	// q is a string
	if q, ok := paramByName["q"]; !ok {
		t.Fatal("expected 'q' query parameter")
	} else if !q.Schema.Value.Type.Is("string") {
		t.Errorf("expected q type 'string', got %v", q.Schema.Value.Type)
	}

	// page is an integer
	if page, ok := paramByName["page"]; !ok {
		t.Fatal("expected 'page' query parameter")
	} else if !page.Schema.Value.Type.Is("integer") {
		t.Errorf("expected page type 'integer', got %v", page.Schema.Value.Type)
	}

	// limit is an integer
	if limit, ok := paramByName["limit"]; !ok {
		t.Fatal("expected 'limit' query parameter")
	} else if !limit.Schema.Value.Type.Is("integer") {
		t.Errorf("expected limit type 'integer', got %v", limit.Schema.Value.Type)
	}
}

func TestSpecQueryParamRequired(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/search", func(r *http.Request, in SearchQuery) (*SearchResult, error) {
		return &SearchResult{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/search").Get

	paramByName := make(map[string]*openapi3.Parameter)
	for _, p := range op.Parameters {
		paramByName[p.Value.Name] = p.Value
	}

	// q has validate:"required" so it should be required
	if !paramByName["q"].Required {
		t.Error("expected 'q' to be required")
	}
	// page does not have required tag
	if paramByName["page"].Required {
		t.Error("expected 'page' to not be required")
	}
}

func TestSpecQueryParamValidationConstraints(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/search", func(r *http.Request, in SearchQuery) (*SearchResult, error) {
		return &SearchResult{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/search").Get

	paramByName := make(map[string]*openapi3.Parameter)
	for _, p := range op.Parameters {
		paramByName[p.Value.Name] = p.Value
	}

	// page has min=1
	pageSchema := paramByName["page"].Schema.Value
	if pageSchema.Min == nil || *pageSchema.Min != 1 {
		t.Errorf("expected page minimum 1, got %v", pageSchema.Min)
	}

	// limit has min=1,max=100
	limitSchema := paramByName["limit"].Schema.Value
	if limitSchema.Min == nil || *limitSchema.Min != 1 {
		t.Errorf("expected limit minimum 1, got %v", limitSchema.Min)
	}
	if limitSchema.Max == nil || *limitSchema.Max != 100 {
		t.Errorf("expected limit maximum 100, got %v", limitSchema.Max)
	}
}

func TestSpecQueryParamEnum(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/filter", func(r *http.Request, in FilterQuery) (*Empty, error) {
		return &Empty{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/filter").Get

	var statusParam *openapi3.Parameter
	for _, p := range op.Parameters {
		if p.Value.Name == "status" {
			statusParam = p.Value
			break
		}
	}
	if statusParam == nil {
		t.Fatal("expected 'status' query parameter")
	}
	if len(statusParam.Schema.Value.Enum) != 3 {
		t.Fatalf("expected 3 enum values, got %d", len(statusParam.Schema.Value.Enum))
	}
}

func TestSpecQueryParamSliceType(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/tags", func(r *http.Request, in TagQuery) (*TagResult, error) {
		return &TagResult{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/tags").Get

	var tagParam *openapi3.Parameter
	for _, p := range op.Parameters {
		if p.Value.Name == "tag" {
			tagParam = p.Value
			break
		}
	}
	if tagParam == nil {
		t.Fatal("expected 'tag' query parameter")
	}
	if !tagParam.Schema.Value.Type.Is("array") {
		t.Errorf("expected tag type 'array', got %v", tagParam.Schema.Value.Type)
	}
	if tagParam.Schema.Value.Items == nil || !tagParam.Schema.Value.Items.Value.Type.Is("string") {
		t.Error("expected tag items type 'string'")
	}
}

func TestSpecQueryParamsCombinedWithPathParams(t *testing.T) {
	api := newTestAPI(t)

	type ItemQuery struct {
		Fields string `query:"fields"`
	}

	shiftapi.Get(api, "/items/{id}", func(r *http.Request, in ItemQuery) (*Item, error) {
		return &Item{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/items/{id}").Get

	pathParams := 0
	queryParams := 0
	for _, p := range op.Parameters {
		switch p.Value.In {
		case "path":
			pathParams++
		case "query":
			queryParams++
		}
	}
	if pathParams != 1 {
		t.Errorf("expected 1 path parameter, got %d", pathParams)
	}
	if queryParams != 1 {
		t.Errorf("expected 1 query parameter, got %d", queryParams)
	}
}

func TestSpecPostWithQueryHasQueryParamsAndBody(t *testing.T) {
	api := newTestAPI(t)

	type CreateInput struct {
		DryRun bool   `query:"dry_run"`
		Name   string `json:"name"`
		ID     string `json:"id"`
	}

	shiftapi.Post(api, "/items", func(r *http.Request, in CreateInput) (*Item, error) {
		return &Item{ID: in.ID, Name: in.Name}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/items").Post
	if op == nil {
		t.Fatal("expected POST operation on /items")
	}

	// Should have query params
	queryParams := 0
	for _, p := range op.Parameters {
		if p.Value.In == "query" {
			queryParams++
		}
	}
	if queryParams != 1 {
		t.Errorf("expected 1 query parameter, got %d", queryParams)
	}

	// Should also have a request body
	if op.RequestBody == nil {
		t.Error("expected request body on POST with query and body")
	}
}

// --- Query-only input should not have request body ---

func TestSpecQueryOnlyInputHasNoRequestBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/search", func(r *http.Request, in SearchQuery) (*SearchResult, error) {
		return &SearchResult{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/search").Get
	if op.RequestBody != nil {
		t.Error("GET with query-only input should not have a request body in the spec")
	}
}

// --- Form upload test types ---

type UploadInput struct {
	File  *multipart.FileHeader `form:"file" validate:"required"`
	Title string                `form:"title" validate:"required"`
}

type UploadResult struct {
	Filename string `json:"filename"`
	Title    string `json:"title"`
	Size     int64  `json:"size"`
}

type MultiUploadInput struct {
	Files []*multipart.FileHeader `form:"files" validate:"required"`
}

type MultiUploadResult struct {
	Count int `json:"count"`
}

type FormWithQueryInput struct {
	File *multipart.FileHeader `form:"file"`
	Tags string                `query:"tags"`
}

type FormWithQueryResult struct {
	HasFile bool   `json:"has_file"`
	Tags    string `json:"tags"`
}

type FormTextFieldsInput struct {
	Name  string  `form:"name"`
	Age   int     `form:"age"`
	Score float64 `form:"score"`
	Admin bool    `form:"admin"`
}

type FormTextFieldsResult struct {
	Name  string  `json:"name"`
	Age   int     `json:"age"`
	Score float64 `json:"score"`
	Admin bool    `json:"admin"`
}

type MixedJsonFormInput struct {
	Name string                `json:"name"`
	File *multipart.FileHeader `form:"file"`
}

type AcceptUploadInput struct {
	Image *multipart.FileHeader `form:"image" accept:"image/png,image/jpeg" validate:"required"`
}

type AcceptMultiUploadInput struct {
	Images []*multipart.FileHeader `form:"images" accept:"image/png,image/jpeg"`
}

// --- Form upload helpers ---

func doMultipartRequest(t *testing.T, api http.Handler, method, path string, fields map[string]string, files map[string][]byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for name, content := range files {
		part, err := w.CreateFormFile(name, name+".txt")
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatalf("failed to write file content: %v", err)
		}
	}
	for name, value := range fields {
		if err := w.WriteField(name, value); err != nil {
			t.Fatalf("failed to write field: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	return rec.Result()
}

// doMultipartRequestMultiFiles sends a multipart request with multiple files under the same field name.
func doMultipartRequestMultiFiles(t *testing.T, api http.Handler, method, path, fieldName string, fileContents [][]byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for i, content := range fileContents {
		part, err := w.CreateFormFile(fieldName, fmt.Sprintf("file%d.txt", i))
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatalf("failed to write file content: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	return rec.Result()
}

// --- Form upload runtime tests ---

func TestPostFormUpload(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/upload", func(r *http.Request, in UploadInput) (*UploadResult, error) {
		return &UploadResult{
			Filename: in.File.Filename,
			Title:    in.Title,
			Size:     in.File.Size,
		}, nil
	})

	resp := doMultipartRequest(t, api, http.MethodPost, "/upload",
		map[string]string{"title": "My Document"},
		map[string][]byte{"file": []byte("hello world")},
	)
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	result := decodeJSON[UploadResult](t, resp)
	if result.Filename != "file.txt" {
		t.Errorf("expected filename %q, got %q", "file.txt", result.Filename)
	}
	if result.Title != "My Document" {
		t.Errorf("expected title %q, got %q", "My Document", result.Title)
	}
	if result.Size != 11 {
		t.Errorf("expected size 11, got %d", result.Size)
	}
}

func TestPostFormUploadMissingRequired(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/upload", func(r *http.Request, in UploadInput) (*UploadResult, error) {
		return &UploadResult{}, nil
	})

	// Send without the required file field
	resp := doMultipartRequest(t, api, http.MethodPost, "/upload",
		map[string]string{"title": "My Document"},
		nil,
	)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		body := readBody(t, resp)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, body)
	}
}

func TestPostFormUploadMultipleFiles(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/upload-multi", func(r *http.Request, in MultiUploadInput) (*MultiUploadResult, error) {
		return &MultiUploadResult{Count: len(in.Files)}, nil
	})

	resp := doMultipartRequestMultiFiles(t, api, http.MethodPost, "/upload-multi", "files",
		[][]byte{[]byte("file1"), []byte("file2"), []byte("file3")},
	)
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	result := decodeJSON[MultiUploadResult](t, resp)
	if result.Count != 3 {
		t.Errorf("expected count 3, got %d", result.Count)
	}
}

func TestPostFormWithQueryParams(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/upload-tags", func(r *http.Request, in FormWithQueryInput) (*FormWithQueryResult, error) {
		return &FormWithQueryResult{
			HasFile: in.File != nil,
			Tags:    in.Tags,
		}, nil
	})

	resp := doMultipartRequest(t, api, http.MethodPost, "/upload-tags?tags=a,b,c",
		nil,
		map[string][]byte{"file": []byte("data")},
	)
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	result := decodeJSON[FormWithQueryResult](t, resp)
	if !result.HasFile {
		t.Error("expected HasFile=true")
	}
	if result.Tags != "a,b,c" {
		t.Errorf("expected tags %q, got %q", "a,b,c", result.Tags)
	}
}

func TestPostFormTextFieldTypes(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/form-fields", func(r *http.Request, in FormTextFieldsInput) (*FormTextFieldsResult, error) {
		return &FormTextFieldsResult{
			Name:  in.Name,
			Age:   in.Age,
			Score: in.Score,
			Admin: in.Admin,
		}, nil
	})

	resp := doMultipartRequest(t, api, http.MethodPost, "/form-fields",
		map[string]string{
			"name":  "Alice",
			"age":   "30",
			"score": "9.5",
			"admin": "true",
		},
		nil,
	)
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	result := decodeJSON[FormTextFieldsResult](t, resp)
	if result.Name != "Alice" {
		t.Errorf("expected name %q, got %q", "Alice", result.Name)
	}
	if result.Age != 30 {
		t.Errorf("expected age 30, got %d", result.Age)
	}
	if result.Score != 9.5 {
		t.Errorf("expected score 9.5, got %f", result.Score)
	}
	if !result.Admin {
		t.Error("expected admin=true")
	}
}

// --- Form upload OpenAPI spec tests ---

func TestSpecFormUploadContentType(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/upload", func(r *http.Request, in UploadInput) (*UploadResult, error) {
		return nil, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/upload").Post
	if op.RequestBody == nil {
		t.Fatal("expected request body")
	}
	if op.RequestBody.Value.Content.Get("multipart/form-data") == nil {
		t.Error("expected multipart/form-data content type")
	}
	if op.RequestBody.Value.Content.Get("application/json") != nil {
		t.Error("should not have application/json content type for form upload")
	}
}

func TestSpecFormUploadFileIsBinary(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/upload", func(r *http.Request, in UploadInput) (*UploadResult, error) {
		return nil, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/upload").Post
	formContent := op.RequestBody.Value.Content.Get("multipart/form-data")
	fileSchema := formContent.Schema.Value.Properties["file"]
	if fileSchema == nil {
		t.Fatal("expected file property in form schema")
	}
	if !fileSchema.Value.Type.Is("string") {
		t.Errorf("expected type string, got %v", fileSchema.Value.Type)
	}
	if fileSchema.Value.Format != "binary" {
		t.Errorf("expected format binary, got %q", fileSchema.Value.Format)
	}
}

func TestSpecFormUploadArrayIsBinaryArray(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/upload-multi", func(r *http.Request, in MultiUploadInput) (*MultiUploadResult, error) {
		return nil, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/upload-multi").Post
	formContent := op.RequestBody.Value.Content.Get("multipart/form-data")
	filesSchema := formContent.Schema.Value.Properties["files"]
	if filesSchema == nil {
		t.Fatal("expected files property in form schema")
	}
	if !filesSchema.Value.Type.Is("array") {
		t.Errorf("expected type array, got %v", filesSchema.Value.Type)
	}
	if filesSchema.Value.Items == nil {
		t.Fatal("expected items in array schema")
	}
	if !filesSchema.Value.Items.Value.Type.Is("string") {
		t.Errorf("expected items type string, got %v", filesSchema.Value.Items.Value.Type)
	}
	if filesSchema.Value.Items.Value.Format != "binary" {
		t.Errorf("expected items format binary, got %q", filesSchema.Value.Items.Value.Format)
	}
}

func TestSpecFormUploadExcludesQueryFields(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/upload-tags", func(r *http.Request, in FormWithQueryInput) (*FormWithQueryResult, error) {
		return nil, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/upload-tags").Post
	formContent := op.RequestBody.Value.Content.Get("multipart/form-data")
	schema := formContent.Schema.Value

	// file should be in form schema
	if schema.Properties["file"] == nil {
		t.Error("expected file property in form schema")
	}
	// tags should NOT be in form schema (it's a query param)
	if schema.Properties["tags"] != nil {
		t.Error("query field 'tags' should not appear in form schema")
	}

	// tags should be a query parameter
	var foundTags bool
	for _, p := range op.Parameters {
		if p.Value.Name == "tags" && p.Value.In == "query" {
			foundTags = true
		}
	}
	if !foundTags {
		t.Error("expected 'tags' as query parameter")
	}
}

func TestSpecFormUploadRequired(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/upload", func(r *http.Request, in UploadInput) (*UploadResult, error) {
		return nil, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/upload").Post
	formContent := op.RequestBody.Value.Content.Get("multipart/form-data")
	required := formContent.Schema.Value.Required

	if !slices.Contains(required, "file") {
		t.Error("expected 'file' in required list")
	}
	if !slices.Contains(required, "title") {
		t.Error("expected 'title' in required list")
	}
}

// --- Form upload edge case tests ---

func TestMixedJsonAndFormTagsPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when mixing json and form tags")
		}
		msg := fmt.Sprintf("%v", r)
		if !strings.Contains(msg, "json and form tags") {
			t.Errorf("expected panic message about mixed tags, got: %s", msg)
		}
	}()

	api := newTestAPI(t)
	shiftapi.Post(api, "/bad", func(r *http.Request, in MixedJsonFormInput) (*Empty, error) {
		return &Empty{}, nil
	})
}

func TestFormUploadEmptyBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/upload", func(r *http.Request, in UploadInput) (*UploadResult, error) {
		return &UploadResult{}, nil
	})

	// Send request with no multipart body
	req := httptest.NewRequest(http.MethodPost, "/upload", nil)
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	resp := rec.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}
}

// --- Accept tag helpers ---

// doMultipartRequestWithContentType sends a multipart request with a file part that has a specific Content-Type.
func doMultipartRequestWithContentType(t *testing.T, api http.Handler, method, path, fieldName, fileName, contentType string, content []byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fileName))
	h.Set("Content-Type", contentType)
	part, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("failed to create part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	return rec.Result()
}

// --- Accept tag runtime tests ---

func TestPostFormAcceptAllowed(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/upload-image", func(r *http.Request, in AcceptUploadInput) (*UploadResult, error) {
		return &UploadResult{Filename: in.Image.Filename}, nil
	})

	resp := doMultipartRequestWithContentType(t, api, http.MethodPost, "/upload-image",
		"image", "photo.png", "image/png", []byte("fake png data"),
	)
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	result := decodeJSON[UploadResult](t, resp)
	if result.Filename != "photo.png" {
		t.Errorf("expected filename %q, got %q", "photo.png", result.Filename)
	}
}

func TestPostFormAcceptRejected(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/upload-image", func(r *http.Request, in AcceptUploadInput) (*UploadResult, error) {
		return &UploadResult{}, nil
	})

	resp := doMultipartRequestWithContentType(t, api, http.MethodPost, "/upload-image",
		"image", "doc.pdf", "application/pdf", []byte("fake pdf data"),
	)
	if resp.StatusCode != http.StatusBadRequest {
		body := readBody(t, resp)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, body)
	}
}

func TestPostFormAcceptMultipleFilesRejected(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/upload-images", func(r *http.Request, in AcceptMultiUploadInput) (*MultiUploadResult, error) {
		return &MultiUploadResult{Count: len(in.Images)}, nil
	})

	// Send a file with wrong content type
	resp := doMultipartRequestWithContentType(t, api, http.MethodPost, "/upload-images",
		"images", "doc.pdf", "application/pdf", []byte("fake pdf data"),
	)
	if resp.StatusCode != http.StatusBadRequest {
		body := readBody(t, resp)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, body)
	}
}

// --- Accept tag OpenAPI spec tests ---

func TestSpecFormAcceptEncoding(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/upload-image", func(r *http.Request, in AcceptUploadInput) (*UploadResult, error) {
		return nil, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/upload-image").Post
	formContent := op.RequestBody.Value.Content.Get("multipart/form-data")

	if formContent.Encoding == nil {
		t.Fatal("expected encoding map to be set")
	}
	enc, ok := formContent.Encoding["image"]
	if !ok {
		t.Fatal("expected encoding entry for 'image'")
	}
	if enc.ContentType != "image/png,image/jpeg" {
		t.Errorf("expected contentType %q, got %q", "image/png,image/jpeg", enc.ContentType)
	}
}

func TestSpecFormNoAcceptNoEncoding(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/upload", func(r *http.Request, in UploadInput) (*UploadResult, error) {
		return nil, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/upload").Post
	formContent := op.RequestBody.Value.Content.Get("multipart/form-data")

	if formContent.Encoding != nil {
		t.Error("expected no encoding map when no accept tags")
	}
}

// --- Required field inference tests ---

type RequiredInferenceResponse struct {
	Name    string  `json:"name"`
	Age     int     `json:"age"`
	Nickname *string `json:"nickname"`
	Bio     *string `json:"bio" validate:"required"`
}

func TestSpecNonPointerFieldsAreRequired(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/user", func(r *http.Request, _ struct{}) (*RequiredInferenceResponse, error) {
		return nil, nil
	})

	spec := api.Spec()
	schemaRef, ok := spec.Components.Schemas["RequiredInferenceResponse"]
	if !ok {
		t.Fatal("expected RequiredInferenceResponse in component schemas")
	}
	schema := schemaRef.Value

	// Non-pointer fields should be required
	if !slices.Contains(schema.Required, "name") {
		t.Errorf("expected 'name' (string) in required, got %v", schema.Required)
	}
	if !slices.Contains(schema.Required, "age") {
		t.Errorf("expected 'age' (int) in required, got %v", schema.Required)
	}

	// Pointer field without validate:"required" should NOT be required
	if slices.Contains(schema.Required, "nickname") {
		t.Errorf("expected 'nickname' (*string) to not be required, got %v", schema.Required)
	}

	// Pointer field with validate:"required" should be required
	if !slices.Contains(schema.Required, "bio") {
		t.Errorf("expected 'bio' (*string validate:required) in required, got %v", schema.Required)
	}
}

type NestedAddress struct {
	Street string  `json:"street"`
	Zip    *string `json:"zip"`
}

type NestedPerson struct {
	Name    string        `json:"name"`
	Address NestedAddress `json:"address"`
}

func TestSpecNestedStructFieldsAreRequired(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/person", func(r *http.Request, _ struct{}) (*NestedPerson, error) {
		return nil, nil
	})

	spec := api.Spec()
	schemaRef, ok := spec.Components.Schemas["NestedPerson"]
	if !ok {
		t.Fatal("expected NestedPerson in component schemas")
	}
	schema := schemaRef.Value

	// Top-level non-pointer fields required
	if !slices.Contains(schema.Required, "name") {
		t.Errorf("expected 'name' in required, got %v", schema.Required)
	}
	if !slices.Contains(schema.Required, "address") {
		t.Errorf("expected 'address' in required, got %v", schema.Required)
	}

	// Nested struct fields
	addrProp := schema.Properties["address"]
	if addrProp == nil || addrProp.Value == nil {
		t.Fatal("expected 'address' property with inline schema")
	}
	addrSchema := addrProp.Value

	if !slices.Contains(addrSchema.Required, "street") {
		t.Errorf("expected nested 'street' (string) in required, got %v", addrSchema.Required)
	}
	if slices.Contains(addrSchema.Required, "zip") {
		t.Errorf("expected nested 'zip' (*string) to not be required, got %v", addrSchema.Required)
	}
}

type OptionalAddress struct {
	Home    *NestedAddress `json:"home"`
	Work    *NestedAddress `json:"work" validate:"required"`
}

func TestSpecPointerStructFieldRequired(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/addrs", func(r *http.Request, _ struct{}) (*OptionalAddress, error) {
		return nil, nil
	})

	spec := api.Spec()
	schemaRef, ok := spec.Components.Schemas["OptionalAddress"]
	if !ok {
		t.Fatal("expected OptionalAddress in component schemas")
	}
	schema := schemaRef.Value

	// *NestedAddress without validate:"required" should NOT be required
	if slices.Contains(schema.Required, "home") {
		t.Errorf("expected 'home' (*NestedAddress) to not be required, got %v", schema.Required)
	}

	// *NestedAddress with validate:"required" should be required
	if !slices.Contains(schema.Required, "work") {
		t.Errorf("expected 'work' (*NestedAddress validate:required) in required, got %v", schema.Required)
	}
}

// --- WithError tests ---

type ConflictError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *ConflictError) Error() string { return e.Message }

type NotFoundError struct {
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

func (e *NotFoundError) Error() string { return e.Message }

func TestErrorComponentSchemasExist(t *testing.T) {
	api := newTestAPI(t)
	spec := api.Spec()

	if _, ok := spec.Components.Schemas["InternalServerError"]; !ok {
		t.Error("expected APIError in component schemas")
	}
	if _, ok := spec.Components.Schemas["ValidationError"]; !ok {
		t.Error("expected ValidationError in component schemas")
	}
}

func TestWithErrorAddsResponseToSpec(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/users/{id}", func(r *http.Request, _ struct{}) (*Item, error) {
		return nil, nil
	}, shiftapi.WithError[*NotFoundError](http.StatusNotFound))

	spec := api.Spec()
	op := spec.Paths.Find("/users/{id}").Get

	resp404 := op.Responses.Value("404")
	if resp404 == nil {
		t.Fatal("expected 404 response in spec")
	}
	if resp404.Value.Description == nil || *resp404.Value.Description != "Not Found" {
		t.Error("expected 404 response description 'Not Found'")
	}
	content := resp404.Value.Content["application/json"]
	if content == nil {
		t.Fatal("expected application/json content in 404 response")
	}
	if content.Schema.Ref != "#/components/schemas/NotFoundError" {
		t.Errorf("expected 404 schema ref to NotFoundError, got %s", content.Schema.Ref)
	}

	// Verify the schema was registered in components
	if _, ok := spec.Components.Schemas["NotFoundError"]; !ok {
		t.Error("expected NotFoundError in component schemas")
	}
}

func TestWithErrorCustomTypeSchema(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/items", func(r *http.Request, in *Item) (*Item, error) {
		return in, nil
	}, shiftapi.WithError[*ConflictError](http.StatusConflict))

	spec := api.Spec()
	schema, ok := spec.Components.Schemas["ConflictError"]
	if !ok {
		t.Fatal("expected ConflictError in component schemas")
	}
	if schema.Value.Properties["code"] == nil {
		t.Error("expected 'code' property in ConflictError schema")
	}
	if schema.Value.Properties["message"] == nil {
		t.Error("expected 'message' property in ConflictError schema")
	}
}

func TestWithErrorMultipleTypes(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/items", func(r *http.Request, in *Item) (*Item, error) {
		return in, nil
	},
		shiftapi.WithError[*NotFoundError](http.StatusNotFound),
		shiftapi.WithError[*ConflictError](http.StatusConflict),
	)

	spec := api.Spec()
	op := spec.Paths.Find("/items").Post
	if op.Responses.Value("404") == nil {
		t.Error("expected 404 response")
	}
	if op.Responses.Value("409") == nil {
		t.Error("expected 409 response")
	}
	if op.Responses.Value("422") == nil {
		t.Error("expected 422 response")
	}
	if op.Responses.Value("500") == nil {
		t.Error("expected 500 response")
	}
}

func TestWithErrorRuntimeMatching(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/items/{id}", func(r *http.Request, _ struct{}) (*Item, error) {
		return nil, &NotFoundError{Message: "not found", Detail: "no item with that ID"}
	}, shiftapi.WithError[*NotFoundError](http.StatusNotFound))

	resp := doRequest(t, api, "GET", "/items/123", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	body := decodeJSON[NotFoundError](t, resp)
	if body.Message != "not found" {
		t.Errorf("expected message 'not found', got %q", body.Message)
	}
	if body.Detail != "no item with that ID" {
		t.Errorf("expected detail 'no item with that ID', got %q", body.Detail)
	}
}

func TestWithErrorWrappedError(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/items/{id}", func(r *http.Request, _ struct{}) (*Item, error) {
		return nil, fmt.Errorf("db lookup failed: %w", &NotFoundError{Message: "not found", Detail: "wrapped"})
	}, shiftapi.WithError[*NotFoundError](http.StatusNotFound))

	resp := doRequest(t, api, "GET", "/items/123", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	body := decodeJSON[NotFoundError](t, resp)
	if body.Detail != "wrapped" {
		t.Errorf("expected detail 'wrapped', got %q", body.Detail)
	}
}

func TestWithErrorUnregisteredFallsTo500(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/items/{id}", func(r *http.Request, _ struct{}) (*Item, error) {
		return nil, &ConflictError{Code: "CONFLICT", Message: "conflict"}
	}) // No WithError registered for ConflictError

	resp := doRequest(t, api, "GET", "/items/123", "")
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 for unregistered error type, got %d", resp.StatusCode)
	}
	body := decodeJSON[struct{ Message string }](t, resp)
	if body.Message != "internal server error" {
		t.Errorf("expected generic message, got %q", body.Message)
	}
}

type ValueReceiverError struct {
	Message string `json:"message"`
}

func (e ValueReceiverError) Error() string { return e.Message }

func TestWithErrorValueReceiver(t *testing.T) {
	// WithError[ValueReceiverError] (non-pointer) should normalize to pointer internally
	api := newTestAPI(t)
	shiftapi.Get(api, "/items", func(r *http.Request, _ struct{}) (*Item, error) {
		return nil, ValueReceiverError{Message: "value receiver error"}
	}, shiftapi.WithError[ValueReceiverError](http.StatusConflict))

	resp := doRequest(t, api, "GET", "/items", "")
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestAllRoutesHave422And500(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/a", func(r *http.Request, _ struct{}) (*Status, error) { return nil, nil })
	shiftapi.Post(api, "/b", func(r *http.Request, _ struct{}) (*Status, error) { return nil, nil })
	shiftapi.Delete(api, "/c", func(r *http.Request, _ struct{}) (*Status, error) { return nil, nil })

	spec := api.Spec()
	for _, tc := range []struct {
		path   string
		method string
	}{
		{"/a", "GET"},
		{"/b", "POST"},
		{"/c", "DELETE"},
	} {
		pathItem := spec.Paths.Find(tc.path)
		var op *openapi3.Operation
		switch tc.method {
		case "GET":
			op = pathItem.Get
		case "POST":
			op = pathItem.Post
		case "DELETE":
			op = pathItem.Delete
		}
		if op.Responses.Value("422") == nil {
			t.Errorf("%s %s: expected 422 response", tc.method, tc.path)
		}
		if op.Responses.Value("500") == nil {
			t.Errorf("%s %s: expected 500 response", tc.method, tc.path)
		}
	}
}

// --- WithInternalServerError tests ---

type CustomServerError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *CustomServerError) Error() string { return e.Message }

func TestWithInternalServerErrorCustomResponse(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithInternalServerError(func(_ error) *CustomServerError {
			return &CustomServerError{
				Code:    "INTERNAL_ERROR",
				Message: "something went wrong",
			}
		}),
	)
	shiftapi.Get(api, "/boom", func(r *http.Request, _ struct{}) (*Empty, error) {
		return nil, errors.New("unexpected")
	})

	resp := doRequest(t, api, "GET", "/boom", "")
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
	body := decodeJSON[CustomServerError](t, resp)
	if body.Code != "INTERNAL_ERROR" {
		t.Errorf("expected code 'INTERNAL_ERROR', got %q", body.Code)
	}
	if body.Message != "something went wrong" {
		t.Errorf("expected message 'something went wrong', got %q", body.Message)
	}
}

func TestWithInternalServerErrorReceivesError(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithInternalServerError(func(err error) *CustomServerError {
			return &CustomServerError{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			}
		}),
	)
	shiftapi.Get(api, "/boom", func(r *http.Request, _ struct{}) (*Empty, error) {
		return nil, errors.New("db connection lost")
	})

	resp := doRequest(t, api, "GET", "/boom", "")
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
	body := decodeJSON[CustomServerError](t, resp)
	if body.Message != "db connection lost" {
		t.Errorf("expected original error message, got %q", body.Message)
	}
}

func TestWithInternalServerErrorSchemaInSpec(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithInternalServerError(func(_ error) *CustomServerError {
			return &CustomServerError{Code: "INTERNAL_ERROR", Message: "something went wrong"}
		}),
	)
	shiftapi.Get(api, "/test", func(r *http.Request, _ struct{}) (*Empty, error) {
		return nil, nil
	})

	spec := api.Spec()
	schema, ok := spec.Components.Schemas["InternalServerError"]
	if !ok {
		t.Fatal("expected InternalServerError in component schemas")
	}
	if schema.Value.Properties["code"] == nil {
		t.Error("expected 'code' property in InternalServerError schema")
	}
	if schema.Value.Properties["message"] == nil {
		t.Error("expected 'message' property in InternalServerError schema")
	}
}

func TestWithInternalServerErrorDefault500StillWorks(t *testing.T) {
	api := shiftapi.New() // no WithInternalServerError
	shiftapi.Get(api, "/boom", func(r *http.Request, _ struct{}) (*Empty, error) {
		return nil, errors.New("unexpected")
	})

	resp := doRequest(t, api, "GET", "/boom", "")
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
	body := decodeJSON[struct{ Message string }](t, resp)
	if body.Message != "internal server error" {
		t.Errorf("expected default message, got %q", body.Message)
	}
}

// --- WithBadRequestError tests ---

type CustomBadRequestError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func TestWithBadRequestErrorCustomResponse(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithBadRequestError(func(err error) *CustomBadRequestError {
			return &CustomBadRequestError{
				Code:    "BAD_REQUEST",
				Message: err.Error(),
			}
		}),
	)
	shiftapi.Post(api, "/test", func(r *http.Request, in struct {
		Name string `json:"name"`
	}) (*Empty, error) {
		return &Empty{}, nil
	})

	resp := doRequest(t, api, "POST", "/test", "not json")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	body := decodeJSON[CustomBadRequestError](t, resp)
	if body.Code != "BAD_REQUEST" {
		t.Errorf("expected code 'BAD_REQUEST', got %q", body.Code)
	}
}

func TestWithBadRequestErrorSchemaInSpec(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithBadRequestError(func(err error) *CustomBadRequestError {
			return &CustomBadRequestError{Code: "BAD_REQUEST", Message: err.Error()}
		}),
	)
	shiftapi.Get(api, "/test", func(r *http.Request, _ struct{}) (*Empty, error) {
		return nil, nil
	})

	spec := api.Spec()
	schema, ok := spec.Components.Schemas["BadRequestError"]
	if !ok {
		t.Fatal("expected BadRequestError in component schemas")
	}
	if schema.Value.Properties["code"] == nil {
		t.Error("expected 'code' property in BadRequestError schema")
	}
	if schema.Value.Properties["message"] == nil {
		t.Error("expected 'message' property in BadRequestError schema")
	}
}

func TestWithBadRequestErrorDefault400StillWorks(t *testing.T) {
	api := shiftapi.New() // no WithBadRequestError
	shiftapi.Post(api, "/test", func(r *http.Request, in struct {
		Name string `json:"name"`
	}) (*Empty, error) {
		return &Empty{}, nil
	})

	resp := doRequest(t, api, "POST", "/test", "not json")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	body := decodeJSON[struct{ Message string }](t, resp)
	if body.Message != "bad request" {
		t.Errorf("expected default message, got %q", body.Message)
	}
}

// --- Group tests ---

type RateLimitError struct {
	Message string `json:"message"`
}

func (e *RateLimitError) Error() string { return e.Message }

func TestGroupPrefixRouting(t *testing.T) {
	api := shiftapi.New()
	v1 := api.Group("/api/v1")

	shiftapi.Get(v1, "/users", func(r *http.Request, _ struct{}) (*struct {
		Name string `json:"name"`
	}, error) {
		return &struct {
			Name string `json:"name"`
		}{Name: "alice"}, nil
	})

	resp := doRequest(t, api, "GET", "/api/v1/users", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body := decodeJSON[struct{ Name string }](t, resp)
	if body.Name != "alice" {
		t.Errorf("expected 'alice', got %q", body.Name)
	}
}

func TestGroupPrefixInSpec(t *testing.T) {
	api := shiftapi.New()
	v1 := api.Group("/api/v1")

	shiftapi.Get(v1, "/users", func(r *http.Request, _ struct{}) (*Empty, error) {
		return nil, nil
	})

	spec := api.Spec()
	if spec.Paths.Find("/api/v1/users") == nil {
		t.Error("expected /api/v1/users in spec")
	}
}

func TestGroupErrorMerge(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithError[*AuthError](http.StatusUnauthorized),
	)
	v1 := api.Group("/api/v1",
		shiftapi.WithError[*RateLimitError](http.StatusTooManyRequests),
	)

	shiftapi.Get(v1, "/users", func(r *http.Request, _ struct{}) (*Empty, error) {
		return nil, nil
	}, shiftapi.WithError[*NotFoundError](http.StatusNotFound))

	spec := api.Spec()
	op := spec.Paths.Find("/api/v1/users").Get

	if op.Responses.Value("401") == nil {
		t.Error("expected 401 from API-level WithError")
	}
	if op.Responses.Value("429") == nil {
		t.Error("expected 429 from group-level WithError")
	}
	if op.Responses.Value("404") == nil {
		t.Error("expected 404 from route-level WithError")
	}
	if op.Responses.Value("422") == nil {
		t.Error("expected 422 (ValidationError)")
	}
	if op.Responses.Value("500") == nil {
		t.Error("expected 500 (InternalServerError)")
	}
}

func TestGroupErrorRuntime(t *testing.T) {
	api := shiftapi.New()
	v1 := api.Group("/api/v1",
		shiftapi.WithError[*RateLimitError](http.StatusTooManyRequests),
	)

	shiftapi.Get(v1, "/users", func(r *http.Request, _ struct{}) (*Empty, error) {
		return nil, &RateLimitError{Message: "slow down"}
	})

	resp := doRequest(t, api, "GET", "/api/v1/users", "")
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", resp.StatusCode)
	}
	body := decodeJSON[RateLimitError](t, resp)
	if body.Message != "slow down" {
		t.Errorf("expected 'slow down', got %q", body.Message)
	}
}

func TestNestedGroups(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithError[*AuthError](http.StatusUnauthorized),
	)
	v1 := api.Group("/api/v1",
		shiftapi.WithError[*RateLimitError](http.StatusTooManyRequests),
	)
	admin := v1.Group("/admin",
		shiftapi.WithError[*ConflictError](http.StatusConflict),
	)

	shiftapi.Get(admin, "/users", func(r *http.Request, _ struct{}) (*Empty, error) {
		return nil, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/api/v1/admin/users").Get

	// Should have errors from all three levels
	if op.Responses.Value("401") == nil {
		t.Error("expected 401 from API-level")
	}
	if op.Responses.Value("429") == nil {
		t.Error("expected 429 from parent group")
	}
	if op.Responses.Value("409") == nil {
		t.Error("expected 409 from nested group")
	}
}

func TestNestedGroupRuntime(t *testing.T) {
	api := shiftapi.New()
	v1 := api.Group("/api/v1")
	admin := v1.Group("/admin")

	shiftapi.Get(admin, "/status", func(r *http.Request, _ struct{}) (*struct {
		OK bool `json:"ok"`
	}, error) {
		return &struct {
			OK bool `json:"ok"`
		}{OK: true}, nil
	})

	resp := doRequest(t, api, "GET", "/api/v1/admin/status", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGroupDoesNotAffectOtherRoutes(t *testing.T) {
	api := shiftapi.New()
	v1 := api.Group("/api/v1",
		shiftapi.WithError[*RateLimitError](http.StatusTooManyRequests),
	)

	// Route on the group
	shiftapi.Get(v1, "/users", func(r *http.Request, _ struct{}) (*Empty, error) {
		return nil, nil
	})

	// Route directly on the API (should NOT have 429)
	shiftapi.Get(api, "/health", func(r *http.Request, _ struct{}) (*Empty, error) {
		return nil, nil
	})

	spec := api.Spec()

	// Group route should have 429
	if spec.Paths.Find("/api/v1/users").Get.Responses.Value("429") == nil {
		t.Error("expected 429 on group route")
	}

	// Direct API route should NOT have 429
	if spec.Paths.Find("/health").Get.Responses.Value("429") != nil {
		t.Error("did not expect 429 on non-group route")
	}
}

func TestGroupTrailingSlash(t *testing.T) {
	api := shiftapi.New()
	g := api.Group("/api/v1/")

	shiftapi.Get(g, "/users", func(r *http.Request, _ struct{}) (*struct {
		Name string `json:"name"`
	}, error) {
		return &struct {
			Name string `json:"name"`
		}{Name: "alice"}, nil
	})

	// Should work at /api/v1/users, not /api/v1//users
	resp := doRequest(t, api, "GET", "/api/v1/users", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGroupPathParamsInPrefix(t *testing.T) {
	api := shiftapi.New()
	g := api.Group("/api/v1/{tenant}")

	shiftapi.Get(g, "/users", func(r *http.Request, _ struct{}) (*struct {
		Tenant string `json:"tenant"`
	}, error) {
		return &struct {
			Tenant string `json:"tenant"`
		}{Tenant: r.PathValue("tenant")}, nil
	})

	resp := doRequest(t, api, "GET", "/api/v1/acme/users", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body := decodeJSON[struct{ Tenant string }](t, resp)
	if body.Tenant != "acme" {
		t.Errorf("expected tenant 'acme', got %q", body.Tenant)
	}

	// Check it appears in the spec too
	spec := api.Spec()
	if spec.Paths.Find("/api/v1/{tenant}/users") == nil {
		t.Error("expected /api/v1/{tenant}/users in spec")
	}
}

// --- API-level WithError (global) tests ---

type AuthError struct {
	Message string `json:"message"`
	Realm   string `json:"realm"`
}

func (e *AuthError) Error() string { return e.Message }

func TestWithErrorGlobalRuntime(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithError[*AuthError](http.StatusUnauthorized),
	)
	shiftapi.Get(api, "/a", func(r *http.Request, _ struct{}) (*Empty, error) {
		return nil, &AuthError{Message: "unauthorized", Realm: "api"}
	})
	shiftapi.Get(api, "/b", func(r *http.Request, _ struct{}) (*Empty, error) {
		return nil, &AuthError{Message: "unauthorized", Realm: "api"}
	})

	for _, path := range []string{"/a", "/b"} {
		resp := doRequest(t, api, "GET", path, "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: expected 401, got %d", path, resp.StatusCode)
		}
		body := decodeJSON[AuthError](t, resp)
		if body.Realm != "api" {
			t.Errorf("%s: expected realm 'api', got %q", path, body.Realm)
		}
	}
}

func TestWithErrorGlobalSpec(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithError[*AuthError](http.StatusUnauthorized),
	)
	shiftapi.Get(api, "/a", func(r *http.Request, _ struct{}) (*Empty, error) { return nil, nil })
	shiftapi.Post(api, "/b", func(r *http.Request, _ struct{}) (*Empty, error) { return nil, nil })

	spec := api.Spec()

	// AuthError should be in component schemas
	if _, ok := spec.Components.Schemas["AuthError"]; !ok {
		t.Error("expected AuthError in component schemas")
	}

	// Both routes should have 401 response
	if spec.Paths.Find("/a").Get.Responses.Value("401") == nil {
		t.Error("GET /a: expected 401 response")
	}
	if spec.Paths.Find("/b").Post.Responses.Value("401") == nil {
		t.Error("POST /b: expected 401 response")
	}
}

func TestWithErrorGlobalAndRouteLevelCombined(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithError[*AuthError](http.StatusUnauthorized),
	)
	shiftapi.Post(api, "/users", func(r *http.Request, _ struct{}) (*Empty, error) {
		return nil, nil
	}, shiftapi.WithError[*ConflictError](http.StatusConflict))

	spec := api.Spec()
	op := spec.Paths.Find("/users").Post

	// Should have both API-level (401) and route-level (409) error responses
	if op.Responses.Value("401") == nil {
		t.Error("expected 401 response from API-level WithError")
	}
	if op.Responses.Value("409") == nil {
		t.Error("expected 409 response from route-level WithError")
	}
	if op.Responses.Value("422") == nil {
		t.Error("expected 422 response")
	}
	if op.Responses.Value("500") == nil {
		t.Error("expected 500 response")
	}
}

// --- Typed path parameters ---

func TestPathParamInt(t *testing.T) {
	api := newTestAPI(t)

	type GetInput struct {
		ID int `path:"id"`
	}

	shiftapi.Get(api, "/users/{id}", func(r *http.Request, in GetInput) (*map[string]int, error) {
		return &map[string]int{"id": in.ID}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/users/42", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[map[string]int](t, resp)
	if result["id"] != 42 {
		t.Errorf("expected id=42, got %d", result["id"])
	}
}

func TestPathParamString(t *testing.T) {
	api := newTestAPI(t)

	type GetInput struct {
		Slug string `path:"slug"`
	}

	shiftapi.Get(api, "/posts/{slug}", func(r *http.Request, in GetInput) (*map[string]string, error) {
		return &map[string]string{"slug": in.Slug}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/posts/hello-world", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[map[string]string](t, resp)
	if result["slug"] != "hello-world" {
		t.Errorf("expected slug=hello-world, got %q", result["slug"])
	}
}

func TestPathParamInvalidTypeReturns400(t *testing.T) {
	api := newTestAPI(t)

	type GetInput struct {
		ID int `path:"id"`
	}

	shiftapi.Get(api, "/users/{id}", func(r *http.Request, in GetInput) (*map[string]int, error) {
		return &map[string]int{"id": in.ID}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/users/notanint", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPathParamValidation(t *testing.T) {
	api := newTestAPI(t)

	type GetInput struct {
		ID int `path:"id" validate:"required,gt=0"`
	}

	shiftapi.Get(api, "/users/{id}", func(r *http.Request, in GetInput) (*map[string]int, error) {
		return &map[string]int{"id": in.ID}, nil
	})

	// Valid
	resp := doRequest(t, api, http.MethodGet, "/users/5", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Invalid — gt=0 fails with 0
	resp = doRequest(t, api, http.MethodGet, "/users/0", "")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for validation failure, got %d", resp.StatusCode)
	}
}

func TestPathParamWithQuery(t *testing.T) {
	api := newTestAPI(t)

	type GetInput struct {
		ID     int    `path:"id"`
		Fields string `query:"fields"`
	}

	shiftapi.Get(api, "/users/{id}", func(r *http.Request, in GetInput) (*map[string]any, error) {
		return &map[string]any{"id": in.ID, "fields": in.Fields}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/users/42?fields=name,email", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[map[string]any](t, resp)
	if result["fields"] != "name,email" {
		t.Errorf("expected fields=name,email, got %q", result["fields"])
	}
}

func TestPathParamWithBody(t *testing.T) {
	api := newTestAPI(t)

	type UpdateInput struct {
		ID   int    `path:"id"`
		Name string `json:"name"`
	}

	shiftapi.Put(api, "/users/{id}", func(r *http.Request, in UpdateInput) (*map[string]any, error) {
		return &map[string]any{"id": in.ID, "name": in.Name}, nil
	})

	resp := doRequest(t, api, http.MethodPut, "/users/42", `{"name":"alice"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[map[string]any](t, resp)
	if int(result["id"].(float64)) != 42 {
		t.Errorf("expected id=42, got %v", result["id"])
	}
	if result["name"] != "alice" {
		t.Errorf("expected name=alice, got %q", result["name"])
	}
}

func TestPathParamMismatchPanics(t *testing.T) {
	api := newTestAPI(t)

	type GetInput struct {
		ID int `path:"missing"`
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for mismatched path tag")
		}
		msg := fmt.Sprint(r)
		if !strings.Contains(msg, "missing") {
			t.Errorf("expected panic message to mention 'missing', got %q", msg)
		}
	}()

	shiftapi.Get(api, "/users/{id}", func(r *http.Request, in GetInput) (*map[string]int, error) {
		return &map[string]int{"id": in.ID}, nil
	})
}

func TestPathParamPointerPanics(t *testing.T) {
	api := newTestAPI(t)

	type GetInput struct {
		ID *int `path:"id"`
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for pointer path field")
		}
	}()

	shiftapi.Get(api, "/users/{id}", func(r *http.Request, in GetInput) (*map[string]int, error) {
		return &map[string]int{}, nil
	})
}

func TestPathParamBackwardCompatible(t *testing.T) {
	// Routes with _ struct{} and r.PathValue still work
	api := newTestAPI(t)

	shiftapi.Get(api, "/users/{id}", func(r *http.Request, _ struct{}) (*map[string]string, error) {
		return &map[string]string{"id": r.PathValue("id")}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/users/abc", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[map[string]string](t, resp)
	if result["id"] != "abc" {
		t.Errorf("expected id=abc, got %q", result["id"])
	}
}

// --- Typed path parameter OpenAPI spec tests ---

func TestSpecPathParamIntType(t *testing.T) {
	api := newTestAPI(t)

	type GetInput struct {
		ID int `path:"id"`
	}

	shiftapi.Get(api, "/users/{id}", func(r *http.Request, in GetInput) (*Status, error) {
		return &Status{OK: true}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/users/{id}").Get
	if len(op.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(op.Parameters))
	}
	p := op.Parameters[0].Value
	if p.Name != "id" {
		t.Errorf("expected param name=id, got %q", p.Name)
	}
	if !p.Required {
		t.Error("expected path param to be required")
	}
	if !p.Schema.Value.Type.Is("integer") {
		t.Errorf("expected type=integer, got %v", p.Schema.Value.Type)
	}
}

func TestSpecPathParamFallbackString(t *testing.T) {
	// Without path tag, falls back to string type
	api := newTestAPI(t)

	shiftapi.Get(api, "/users/{id}", func(r *http.Request, _ struct{}) (*Status, error) {
		return &Status{OK: true}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/users/{id}").Get
	p := op.Parameters[0].Value
	if !p.Schema.Value.Type.Is("string") {
		t.Errorf("expected type=string for untagged path param, got %v", p.Schema.Value.Type)
	}
}

func TestSpecPathParamWithValidation(t *testing.T) {
	api := newTestAPI(t)

	type GetInput struct {
		ID string `path:"id" validate:"uuid"`
	}

	shiftapi.Get(api, "/users/{id}", func(r *http.Request, in GetInput) (*Status, error) {
		return &Status{OK: true}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/users/{id}").Get
	p := op.Parameters[0].Value
	if p.Schema.Value.Format != "uuid" {
		t.Errorf("expected format=uuid, got %q", p.Schema.Value.Format)
	}
}

func TestSpecPathParamExcludedFromBody(t *testing.T) {
	api := newTestAPI(t)

	type UpdateInput struct {
		ID   int    `path:"id"`
		Name string `json:"name"`
	}

	shiftapi.Put(api, "/users/{id}", func(r *http.Request, in UpdateInput) (*Status, error) {
		return &Status{OK: true}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/users/{id}").Put

	// Path param should be in parameters
	if len(op.Parameters) != 1 {
		t.Fatalf("expected 1 path parameter, got %d", len(op.Parameters))
	}
	if op.Parameters[0].Value.Name != "id" {
		t.Errorf("expected path param name=id, got %q", op.Parameters[0].Value.Name)
	}

	// Body should have "name" but not "ID"
	body := op.RequestBody.Value.Content["application/json"]
	ref := body.Schema.Ref
	schemaName := ref[len("#/components/schemas/"):]
	schema := spec.Components.Schemas[schemaName].Value
	if _, ok := schema.Properties["name"]; !ok {
		t.Error("expected 'name' in body schema")
	}
	if _, ok := schema.Properties["ID"]; ok {
		t.Error("path field 'ID' should not appear in body schema")
	}
}

// --- Header parameter test types ---

type AuthHeader struct {
	Token string `header:"Authorization" validate:"required"`
}

type AuthResult struct {
	Token string `json:"token"`
}

type OptionalHeader struct {
	Debug *bool `header:"X-Debug"`
	Limit *int  `header:"X-Limit"`
}

type OptionalHeaderResult struct {
	HasDebug bool `json:"has_debug"`
	Debug    bool `json:"debug"`
	HasLimit bool `json:"has_limit"`
	Limit    int  `json:"limit"`
}

type HeaderAndBody struct {
	Token string `header:"Authorization" validate:"required"`
	Name  string `json:"name" validate:"required"`
}

type HeaderAndBodyResult struct {
	Token string `json:"token"`
	Name  string `json:"name"`
}

type HeaderAndQuery struct {
	Token string `header:"Authorization" validate:"required"`
	Q     string `query:"q"`
}

type HeaderAndQueryResult struct {
	Token string `json:"token"`
	Q     string `json:"q"`
}

type ScalarHeaders struct {
	Flag  bool    `header:"X-Flag"`
	Count uint    `header:"X-Count"`
	Score float64 `header:"X-Score"`
}

type ScalarHeaderResult struct {
	Flag  bool    `json:"flag"`
	Count uint    `json:"count"`
	Score float64 `json:"score"`
}

// --- Header parameter test helpers ---

func doRequestWithHeaders(t *testing.T, api http.Handler, method, path, body string, headers map[string]string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	return rec.Result()
}

// --- Header parameter runtime tests ---

func TestGetWithHeaderBasic(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/auth", func(r *http.Request, in AuthHeader) (*AuthResult, error) {
		return &AuthResult{Token: in.Token}, nil
	})

	resp := doRequestWithHeaders(t, api, http.MethodGet, "/auth", "", map[string]string{
		"Authorization": "Bearer abc123",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[AuthResult](t, resp)
	if result.Token != "Bearer abc123" {
		t.Errorf("expected Token=%q, got %q", "Bearer abc123", result.Token)
	}
}

func TestGetWithHeaderMissingRequired(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/auth", func(r *http.Request, in AuthHeader) (*AuthResult, error) {
		return &AuthResult{Token: in.Token}, nil
	})

	// Missing required "Authorization" header
	resp := doRequest(t, api, http.MethodGet, "/auth", "")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestGetWithHeaderInvalidType(t *testing.T) {
	api := newTestAPI(t)
	type IntHeader struct {
		Count int `header:"X-Count" validate:"required"`
	}
	shiftapi.Get(api, "/count", func(r *http.Request, in IntHeader) (*Status, error) {
		return &Status{OK: true}, nil
	})

	resp := doRequestWithHeaders(t, api, http.MethodGet, "/count", "", map[string]string{
		"X-Count": "notanumber",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetWithHeaderOptionalPointers(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/optional", func(r *http.Request, in OptionalHeader) (*OptionalHeaderResult, error) {
		result := &OptionalHeaderResult{}
		if in.Debug != nil {
			result.HasDebug = true
			result.Debug = *in.Debug
		}
		if in.Limit != nil {
			result.HasLimit = true
			result.Limit = *in.Limit
		}
		return result, nil
	})

	// With both headers
	resp := doRequestWithHeaders(t, api, http.MethodGet, "/optional", "", map[string]string{
		"X-Debug": "true",
		"X-Limit": "50",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[OptionalHeaderResult](t, resp)
	if !result.HasDebug || !result.Debug {
		t.Error("expected Debug to be true")
	}
	if !result.HasLimit || result.Limit != 50 {
		t.Errorf("expected Limit=50, got %d", result.Limit)
	}

	// Without optional headers
	resp2 := doRequest(t, api, http.MethodGet, "/optional", "")
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}
	result2 := decodeJSON[OptionalHeaderResult](t, resp2)
	if result2.HasDebug {
		t.Error("expected HasDebug=false when header absent")
	}
	if result2.HasLimit {
		t.Error("expected HasLimit=false when header absent")
	}
}

func TestPostWithHeaderAndBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/items", func(r *http.Request, in HeaderAndBody) (*HeaderAndBodyResult, error) {
		return &HeaderAndBodyResult{Token: in.Token, Name: in.Name}, nil
	})

	resp := doRequestWithHeaders(t, api, http.MethodPost, "/items", `{"name":"widget"}`, map[string]string{
		"Authorization": "Bearer xyz",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[HeaderAndBodyResult](t, resp)
	if result.Token != "Bearer xyz" {
		t.Errorf("expected Token=%q, got %q", "Bearer xyz", result.Token)
	}
	if result.Name != "widget" {
		t.Errorf("expected Name=%q, got %q", "widget", result.Name)
	}
}

func TestHeaderFieldNotSetByBodyDecode(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/items", func(r *http.Request, in HeaderAndBody) (*HeaderAndBodyResult, error) {
		return &HeaderAndBodyResult{Token: in.Token, Name: in.Name}, nil
	})

	// Body includes "Token" key that matches the header field name — it should be ignored
	resp := doRequestWithHeaders(t, api, http.MethodPost, "/items", `{"name":"widget","Token":"body-token"}`, map[string]string{
		"Authorization": "Bearer real",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[HeaderAndBodyResult](t, resp)
	if result.Token != "Bearer real" {
		t.Errorf("expected Token=%q from header, got %q", "Bearer real", result.Token)
	}
}

func TestGetWithHeaderAndQuery(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/search", func(r *http.Request, in HeaderAndQuery) (*HeaderAndQueryResult, error) {
		return &HeaderAndQueryResult{Token: in.Token, Q: in.Q}, nil
	})

	resp := doRequestWithHeaders(t, api, http.MethodGet, "/search?q=hello", "", map[string]string{
		"Authorization": "Bearer abc",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[HeaderAndQueryResult](t, resp)
	if result.Token != "Bearer abc" {
		t.Errorf("expected Token=%q, got %q", "Bearer abc", result.Token)
	}
	if result.Q != "hello" {
		t.Errorf("expected Q=%q, got %q", "hello", result.Q)
	}
}

func TestGetWithHeaderScalars(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/scalars", func(r *http.Request, in ScalarHeaders) (*ScalarHeaderResult, error) {
		return &ScalarHeaderResult{Flag: in.Flag, Count: in.Count, Score: in.Score}, nil
	})

	resp := doRequestWithHeaders(t, api, http.MethodGet, "/scalars", "", map[string]string{
		"X-Flag":  "true",
		"X-Count": "42",
		"X-Score": "3.14",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := decodeJSON[ScalarHeaderResult](t, resp)
	if !result.Flag {
		t.Error("expected Flag=true")
	}
	if result.Count != 42 {
		t.Errorf("expected Count=42, got %d", result.Count)
	}
	if result.Score != 3.14 {
		t.Errorf("expected Score=3.14, got %f", result.Score)
	}
}

func TestGetWithHeaderInvalidBool(t *testing.T) {
	api := newTestAPI(t)
	type BoolHeader struct {
		Flag bool `header:"X-Flag"`
	}
	shiftapi.Get(api, "/test", func(r *http.Request, in BoolHeader) (*Status, error) {
		return &Status{OK: true}, nil
	})

	resp := doRequestWithHeaders(t, api, http.MethodGet, "/test", "", map[string]string{
		"X-Flag": "notabool",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetWithHeaderInvalidUint(t *testing.T) {
	api := newTestAPI(t)
	type UintHeader struct {
		Count uint `header:"X-Count"`
	}
	shiftapi.Get(api, "/test", func(r *http.Request, in UintHeader) (*Status, error) {
		return &Status{OK: true}, nil
	})

	resp := doRequestWithHeaders(t, api, http.MethodGet, "/test", "", map[string]string{
		"X-Count": "-1",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetWithHeaderInvalidFloat(t *testing.T) {
	api := newTestAPI(t)
	type FloatHeader struct {
		Score float64 `header:"X-Score"`
	}
	shiftapi.Get(api, "/test", func(r *http.Request, in FloatHeader) (*Status, error) {
		return &Status{OK: true}, nil
	})

	resp := doRequestWithHeaders(t, api, http.MethodGet, "/test", "", map[string]string{
		"X-Score": "abc",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// --- Header parameter OpenAPI spec tests ---

func TestSpecHeaderParamsDocumented(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/auth", func(r *http.Request, in AuthHeader) (*AuthResult, error) {
		return &AuthResult{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/auth").Get

	var found bool
	for _, p := range op.Parameters {
		if p.Value.Name == "Authorization" && p.Value.In == "header" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Authorization header parameter documented in spec")
	}
}

func TestSpecHeaderParamTypes(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/scalars", func(r *http.Request, in ScalarHeaders) (*ScalarHeaderResult, error) {
		return &ScalarHeaderResult{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/scalars").Get

	expected := map[string]string{
		"X-Flag":  "boolean",
		"X-Count": "integer",
		"X-Score": "number",
	}
	for _, p := range op.Parameters {
		if p.Value.In != "header" {
			continue
		}
		want, ok := expected[p.Value.Name]
		if !ok {
			t.Errorf("unexpected header param %q", p.Value.Name)
			continue
		}
		got := p.Value.Schema.Value.Type.Slice()[0]
		if got != want {
			t.Errorf("header %q: expected type %q, got %q", p.Value.Name, want, got)
		}
	}
}

func TestSpecHeaderParamRequired(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/auth", func(r *http.Request, in AuthHeader) (*AuthResult, error) {
		return &AuthResult{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/auth").Get

	for _, p := range op.Parameters {
		if p.Value.Name == "Authorization" && p.Value.In == "header" {
			if !p.Value.Required {
				t.Error("expected Authorization header to be required")
			}
			return
		}
	}
	t.Error("Authorization header param not found")
}

func TestSpecHeaderParamOptionalPointerNotRequired(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/optional", func(r *http.Request, in OptionalHeader) (*OptionalHeaderResult, error) {
		return &OptionalHeaderResult{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/optional").Get

	for _, p := range op.Parameters {
		if p.Value.In == "header" && p.Value.Required {
			t.Errorf("optional header %q should not be required", p.Value.Name)
		}
	}
}

func TestSpecHeaderParamValidationConstraints(t *testing.T) {
	api := newTestAPI(t)
	type ConstrainedHeader struct {
		Count int `header:"X-Count" validate:"min=1,max=100"`
	}
	shiftapi.Get(api, "/constrained", func(r *http.Request, in ConstrainedHeader) (*Status, error) {
		return &Status{OK: true}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/constrained").Get

	for _, p := range op.Parameters {
		if p.Value.Name == "X-Count" && p.Value.In == "header" {
			s := p.Value.Schema.Value
			if s.Min == nil || *s.Min != 1 {
				t.Error("expected Min=1 on X-Count header param")
			}
			if s.Max == nil || *s.Max != 100 {
				t.Error("expected Max=100 on X-Count header param")
			}
			return
		}
	}
	t.Error("X-Count header param not found")
}

func TestSpecBodySchemaExcludesHeaderFields(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/items", func(r *http.Request, in HeaderAndBody) (*HeaderAndBodyResult, error) {
		return &HeaderAndBodyResult{}, nil
	})

	spec := api.Spec()
	// Find the body schema in components
	for name, schemaRef := range spec.Components.Schemas {
		if name == "HeaderAndBody" {
			if _, has := schemaRef.Value.Properties["Token"]; has {
				t.Error("body schema should not contain header field 'Token'")
			}
			if _, has := schemaRef.Value.Properties["name"]; !has {
				t.Error("body schema should contain body field 'name'")
			}
			return
		}
	}
	t.Error("HeaderAndBody schema not found in components")
}

func TestSpecHeaderParamsCombinedWithQueryParams(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/search", func(r *http.Request, in HeaderAndQuery) (*HeaderAndQueryResult, error) {
		return &HeaderAndQueryResult{}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/search").Get

	var headerParams, queryParams int
	for _, p := range op.Parameters {
		switch p.Value.In {
		case "header":
			headerParams++
		case "query":
			queryParams++
		}
	}
	if headerParams != 1 {
		t.Errorf("expected 1 header param, got %d", headerParams)
	}
	if queryParams != 1 {
		t.Errorf("expected 1 query param, got %d", queryParams)
	}
}

func TestSpecHeaderParamEnum(t *testing.T) {
	api := newTestAPI(t)
	type EnumHeader struct {
		Format string `header:"Accept" validate:"oneof=json xml csv"`
	}
	shiftapi.Get(api, "/data", func(r *http.Request, in EnumHeader) (*Status, error) {
		return &Status{OK: true}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/data").Get

	for _, p := range op.Parameters {
		if p.Value.Name == "Accept" && p.Value.In == "header" {
			if len(p.Value.Schema.Value.Enum) != 3 {
				t.Errorf("expected 3 enum values, got %d", len(p.Value.Schema.Value.Enum))
			}
			return
		}
	}
	t.Error("Accept header param not found")
}

// --- Response header tests ---

type RespWithHeader struct {
	CacheControl string `header:"Cache-Control"`
	Message      string `json:"message"`
}

func TestResponseHeaderIsSet(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/cached", func(r *http.Request, in struct{}) (RespWithHeader, error) {
		return RespWithHeader{CacheControl: "max-age=3600", Message: "hello"}, nil
	})

	resp := doRequest(t, api, "GET", "/cached", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Cache-Control"); got != "max-age=3600" {
		t.Errorf("expected Cache-Control header %q, got %q", "max-age=3600", got)
	}
}

func TestResponseHeaderStrippedFromBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/cached", func(r *http.Request, in struct{}) (RespWithHeader, error) {
		return RespWithHeader{CacheControl: "max-age=3600", Message: "hello"}, nil
	})

	resp := doRequest(t, api, "GET", "/cached", "")
	body := decodeJSON[map[string]any](t, resp)
	if _, exists := body["CacheControl"]; exists {
		t.Error("CacheControl should not appear in response body")
	}
	if _, exists := body["cache-control"]; exists {
		t.Error("cache-control should not appear in response body")
	}
	if msg, ok := body["message"]; !ok || msg != "hello" {
		t.Errorf("expected message %q, got %v", "hello", msg)
	}
}

type RespWithOptionalHeader struct {
	ETag    *string `header:"ETag"`
	Message string  `json:"message"`
}

func TestResponseHeaderOptionalNil(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/item", func(r *http.Request, in struct{}) (RespWithOptionalHeader, error) {
		return RespWithOptionalHeader{Message: "ok"}, nil
	})

	resp := doRequest(t, api, "GET", "/item", "")
	if got := resp.Header.Get("ETag"); got != "" {
		t.Errorf("expected no ETag header, got %q", got)
	}
}

func TestResponseHeaderOptionalSet(t *testing.T) {
	api := newTestAPI(t)
	etag := `"abc123"`
	shiftapi.Get(api, "/item", func(r *http.Request, in struct{}) (RespWithOptionalHeader, error) {
		return RespWithOptionalHeader{ETag: &etag, Message: "ok"}, nil
	})

	resp := doRequest(t, api, "GET", "/item", "")
	if got := resp.Header.Get("ETag"); got != etag {
		t.Errorf("expected ETag header %q, got %q", etag, got)
	}
}

type RespWithIntHeader struct {
	RateLimit int    `header:"X-Rate-Limit"`
	Message   string `json:"message"`
}

func TestResponseHeaderIntType(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/rate", func(r *http.Request, in struct{}) (RespWithIntHeader, error) {
		return RespWithIntHeader{RateLimit: 100, Message: "ok"}, nil
	})

	resp := doRequest(t, api, "GET", "/rate", "")
	if got := resp.Header.Get("X-Rate-Limit"); got != "100" {
		t.Errorf("expected X-Rate-Limit header %q, got %q", "100", got)
	}
}

type RespHeaderOnly struct {
	Location string `header:"Location"`
}

func TestResponseHeaderOnlyNoBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/create", func(r *http.Request, in struct{}) (RespHeaderOnly, error) {
		return RespHeaderOnly{Location: "/items/42"}, nil
	}, shiftapi.WithStatus(http.StatusCreated))

	resp := doRequest(t, api, "POST", "/create", "{}")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/items/42" {
		t.Errorf("expected Location header %q, got %q", "/items/42", got)
	}
}

func TestResponseHeaderOpenAPISchema(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/cached", func(r *http.Request, in struct{}) (RespWithHeader, error) {
		return RespWithHeader{}, nil
	})

	spec := api.Spec()
	pathItem := spec.Paths.Find("/cached")
	if pathItem == nil || pathItem.Get == nil {
		t.Fatal("expected GET /cached in spec")
	}

	respRef := pathItem.Get.Responses.Value("200")
	if respRef == nil || respRef.Value == nil {
		t.Fatal("expected 200 response in spec")
	}

	// Check response header is declared
	if respRef.Value.Headers == nil {
		t.Fatal("expected response headers in spec")
	}
	hdr := respRef.Value.Headers["Cache-Control"]
	if hdr == nil || hdr.Value == nil {
		t.Fatal("expected Cache-Control response header in spec")
	}
	if !hdr.Value.Required {
		t.Error("expected non-pointer response header to be required")
	}
	if hdr.Value.Schema == nil || !hdr.Value.Schema.Value.Type.Is("string") {
		t.Error("expected Cache-Control schema type to be string")
	}

	// Check response body schema does not contain header field
	if respRef.Value.Content != nil {
		jsonMedia := respRef.Value.Content["application/json"]
		if jsonMedia != nil && jsonMedia.Schema != nil && jsonMedia.Schema.Ref != "" {
			// Find the schema in components
			refName := jsonMedia.Schema.Ref[len("#/components/schemas/"):]
			schema := spec.Components.Schemas[refName]
			if schema != nil && schema.Value != nil {
				if _, exists := schema.Value.Properties["CacheControl"]; exists {
					t.Error("CacheControl should not be in response body schema")
				}
			}
		}
	}
}

func TestResponseHeaderOpenAPIOptionalNotRequired(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/item", func(r *http.Request, in struct{}) (RespWithOptionalHeader, error) {
		return RespWithOptionalHeader{}, nil
	})

	spec := api.Spec()
	pathItem := spec.Paths.Find("/item")
	respRef := pathItem.Get.Responses.Value("200")
	hdr := respRef.Value.Headers["Etag"]
	if hdr == nil || hdr.Value == nil {
		t.Fatal("expected ETag response header in spec")
	}
	if hdr.Value.Required {
		t.Error("expected pointer response header to not be required")
	}
}

type RespWithMultipleHeaders struct {
	CacheControl string `header:"Cache-Control"`
	XRequestID   string `header:"X-Request-Id"`
	Message      string `json:"message"`
}

func TestResponseHeaderMultiple(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/multi", func(r *http.Request, in struct{}) (RespWithMultipleHeaders, error) {
		return RespWithMultipleHeaders{
			CacheControl: "no-store",
			XRequestID:   "req-123",
			Message:      "ok",
		}, nil
	})

	resp := doRequest(t, api, "GET", "/multi", "")
	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("expected Cache-Control %q, got %q", "no-store", got)
	}
	if got := resp.Header.Get("X-Request-Id"); got != "req-123" {
		t.Errorf("expected X-Request-Id %q, got %q", "req-123", got)
	}
	body := decodeJSON[map[string]any](t, resp)
	if _, exists := body["CacheControl"]; exists {
		t.Error("CacheControl should not appear in body")
	}
	if _, exists := body["XRequestID"]; exists {
		t.Error("XRequestID should not appear in body")
	}
	if msg, ok := body["message"]; !ok || msg != "ok" {
		t.Errorf("expected message %q, got %v", "ok", msg)
	}
}

func TestResponseHeaderMultipleOpenAPISchema(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/multi", func(r *http.Request, in struct{}) (RespWithMultipleHeaders, error) {
		return RespWithMultipleHeaders{}, nil
	})

	spec := api.Spec()
	respRef := spec.Paths.Find("/multi").Get.Responses.Value("200")
	if respRef.Value.Headers == nil {
		t.Fatal("expected response headers in spec")
	}
	for _, name := range []string{"Cache-Control", "X-Request-Id"} {
		hdr := respRef.Value.Headers[name]
		if hdr == nil || hdr.Value == nil {
			t.Errorf("expected %s response header in spec", name)
		}
	}
}

type RespWithBoolHeader struct {
	Debug   bool   `header:"X-Debug"`
	Message string `json:"message"`
}

func TestResponseHeaderBoolType(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/debug", func(r *http.Request, in struct{}) (RespWithBoolHeader, error) {
		return RespWithBoolHeader{Debug: true, Message: "ok"}, nil
	})

	resp := doRequest(t, api, "GET", "/debug", "")
	if got := resp.Header.Get("X-Debug"); got != "true" {
		t.Errorf("expected X-Debug header %q, got %q", "true", got)
	}
}

func TestResponseHeaderBoolFalseAlwaysSent(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/debug", func(r *http.Request, in struct{}) (RespWithBoolHeader, error) {
		return RespWithBoolHeader{Debug: false, Message: "ok"}, nil
	})

	resp := doRequest(t, api, "GET", "/debug", "")
	// Non-pointer fields are always sent, even when zero.
	if got := resp.Header.Get("X-Debug"); got != "false" {
		t.Errorf("expected X-Debug header %q, got %q", "false", got)
	}
}

func TestResponseHeaderPointerResp(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/ptr", func(r *http.Request, in struct{}) (*RespWithHeader, error) {
		return &RespWithHeader{CacheControl: "private", Message: "ok"}, nil
	})

	resp := doRequest(t, api, "GET", "/ptr", "")
	if got := resp.Header.Get("Cache-Control"); got != "private" {
		t.Errorf("expected Cache-Control %q, got %q", "private", got)
	}
	body := decodeJSON[map[string]any](t, resp)
	if _, exists := body["CacheControl"]; exists {
		t.Error("CacheControl should not appear in body")
	}
}

func TestResponseHeaderWithInputHeaders(t *testing.T) {
	type ReqWithHeader struct {
		Auth string `header:"Authorization"`
	}

	api := newTestAPI(t)
	shiftapi.Get(api, "/echo", func(r *http.Request, in ReqWithHeader) (RespWithHeader, error) {
		return RespWithHeader{CacheControl: "no-cache", Message: "auth=" + in.Auth}, nil
	})

	req := httptest.NewRequest("GET", "/echo", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	result := rec.Result()

	if got := result.Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("expected Cache-Control %q, got %q", "no-cache", got)
	}
	body := decodeJSON[map[string]any](t, result)
	if msg, ok := body["message"]; !ok || msg != "auth=Bearer tok" {
		t.Errorf("expected message %q, got %v", "auth=Bearer tok", msg)
	}
}

func TestResponseHeaderZeroStringAlwaysSent(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/empty", func(r *http.Request, in struct{}) (RespWithHeader, error) {
		return RespWithHeader{Message: "ok"}, nil
	})

	req := httptest.NewRequest("GET", "/empty", nil)
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	// Non-pointer fields are always sent, even when zero.
	// Use the header map directly since Get("") and "not present" both return "".
	if _, exists := rec.Header()["Cache-Control"]; !exists {
		t.Error("expected Cache-Control header to be present (even with empty value)")
	}
}

func TestResponseHeaderWithGroup(t *testing.T) {
	api := newTestAPI(t)
	g := api.Group("/api")
	shiftapi.Get(g, "/item", func(r *http.Request, in struct{}) (RespWithHeader, error) {
		return RespWithHeader{CacheControl: "public", Message: "ok"}, nil
	})

	resp := doRequest(t, api, "GET", "/api/item", "")
	if got := resp.Header.Get("Cache-Control"); got != "public" {
		t.Errorf("expected Cache-Control %q, got %q", "public", got)
	}
}

func TestResponseHeaderOnlyOpenAPINoContent(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/redir", func(r *http.Request, in struct{}) (RespHeaderOnly, error) {
		return RespHeaderOnly{Location: "/new"}, nil
	})

	spec := api.Spec()
	respRef := spec.Paths.Find("/redir").Get.Responses.Value("200")
	if respRef.Value.Headers == nil {
		t.Fatal("expected response headers in spec")
	}
	hdr := respRef.Value.Headers["Location"]
	if hdr == nil || hdr.Value == nil {
		t.Fatal("expected Location response header in spec")
	}
}

func TestResponseHeaderOpenAPIIntegerType(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/rate", func(r *http.Request, in struct{}) (RespWithIntHeader, error) {
		return RespWithIntHeader{}, nil
	})

	spec := api.Spec()
	respRef := spec.Paths.Find("/rate").Get.Responses.Value("200")
	hdr := respRef.Value.Headers["X-Rate-Limit"]
	if hdr == nil || hdr.Value == nil {
		t.Fatal("expected X-Rate-Limit response header in spec")
	}
	if !hdr.Value.Schema.Value.Type.Is("integer") {
		t.Errorf("expected integer type for X-Rate-Limit, got %v", hdr.Value.Schema.Value.Type)
	}
}

func TestResponseHeaderNoHeaderResp(t *testing.T) {
	// Verify that a response without header tags works normally (no regression).
	api := newTestAPI(t)
	shiftapi.Get(api, "/plain", func(r *http.Request, in struct{}) (Greeting, error) {
		return Greeting{Hello: "world"}, nil
	})

	resp := doRequest(t, api, "GET", "/plain", "")
	body := decodeJSON[Greeting](t, resp)
	if body.Hello != "world" {
		t.Errorf("expected Hello %q, got %q", "world", body.Hello)
	}
}

type RespWithOmitempty struct {
	XRequestID string `header:"X-Request-Id"`
	Name       string `json:"name"`
	Nickname   string `json:"nickname,omitempty"`
}

func TestResponseHeaderPreservesOmitempty(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/omit", func(r *http.Request, in struct{}) (RespWithOmitempty, error) {
		return RespWithOmitempty{XRequestID: "req-1", Name: "Alice"}, nil
	})

	resp := doRequest(t, api, "GET", "/omit", "")
	if got := resp.Header.Get("X-Request-Id"); got != "req-1" {
		t.Errorf("expected X-Request-Id %q, got %q", "req-1", got)
	}
	raw := readBody(t, resp)
	// omitempty should suppress the zero-value Nickname field.
	if strings.Contains(raw, "nickname") {
		t.Errorf("expected nickname to be omitted via omitempty, got body: %s", raw)
	}
	if !strings.Contains(raw, `"name":"Alice"`) {
		t.Errorf("expected name in body, got: %s", raw)
	}
	// Header field should not leak into JSON body.
	if strings.Contains(raw, "XRequestID") || strings.Contains(raw, "X-Request-Id") {
		t.Errorf("expected header field stripped from body, got: %s", raw)
	}
}

type respWithCustomMarshal struct {
	XReq    string `header:"X-Req"`
	Message string `json:"message"`
}

func (r respWithCustomMarshal) MarshalJSON() ([]byte, error) {
	return []byte(`{"custom":true}`), nil
}

func TestResponseHeaderWithCustomMarshalJSON(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/custom", func(r *http.Request, in struct{}) (respWithCustomMarshal, error) {
		return respWithCustomMarshal{XReq: "abc", Message: "hello"}, nil
	})

	resp := doRequest(t, api, "GET", "/custom", "")

	// Header should still be extracted and set.
	if got := resp.Header.Get("X-Req"); got != "abc" {
		t.Errorf("expected X-Req header %q, got %q", "abc", got)
	}

	// Custom MarshalJSON should be preserved — body is {"custom":true}.
	raw := readBody(t, resp)
	if !strings.Contains(raw, `"custom":true`) {
		t.Errorf("expected custom MarshalJSON output, got: %s", raw)
	}
}

// --- WithResponseHeader tests ---

func TestWithResponseHeaderRoute(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/secure", func(r *http.Request, in struct{}) (Greeting, error) {
		return Greeting{Hello: "world"}, nil
	}, shiftapi.WithResponseHeader("X-Content-Type-Options", "nosniff"))

	resp := doRequest(t, api, "GET", "/secure", "")
	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("expected X-Content-Type-Options %q, got %q", "nosniff", got)
	}
	// Body should be unaffected.
	body := decodeJSON[Greeting](t, resp)
	if body.Hello != "world" {
		t.Errorf("expected Hello %q, got %q", "world", body.Hello)
	}
}

func TestWithResponseHeaderAPI(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithResponseHeader("X-Frame-Options", "DENY"),
	)
	shiftapi.Get(api, "/a", func(r *http.Request, in struct{}) (Greeting, error) {
		return Greeting{Hello: "a"}, nil
	})
	shiftapi.Get(api, "/b", func(r *http.Request, in struct{}) (Greeting, error) {
		return Greeting{Hello: "b"}, nil
	})

	for _, path := range []string{"/a", "/b"} {
		resp := doRequest(t, api, "GET", path, "")
		if got := resp.Header.Get("X-Frame-Options"); got != "DENY" {
			t.Errorf("%s: expected X-Frame-Options %q, got %q", path, "DENY", got)
		}
	}
}

func TestWithResponseHeaderGroup(t *testing.T) {
	api := shiftapi.New()
	g := api.Group("/api",
		shiftapi.WithResponseHeader("X-API-Version", "2"),
	)
	shiftapi.Get(g, "/item", func(r *http.Request, in struct{}) (Greeting, error) {
		return Greeting{Hello: "ok"}, nil
	})
	// Route outside the group should NOT have the header.
	shiftapi.Get(api, "/health", func(r *http.Request, in struct{}) (Greeting, error) {
		return Greeting{Hello: "ok"}, nil
	})

	resp := doRequest(t, api, "GET", "/api/item", "")
	if got := resp.Header.Get("X-API-Version"); got != "2" {
		t.Errorf("expected X-API-Version %q, got %q", "2", got)
	}

	resp2 := doRequest(t, api, "GET", "/health", "")
	if got := resp2.Header.Get("X-API-Version"); got != "" {
		t.Errorf("expected no X-API-Version on /health, got %q", got)
	}
}

func TestWithResponseHeaderInheritsFromAPIAndGroup(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithResponseHeader("X-Global", "yes"),
	)
	g := api.Group("/v1",
		shiftapi.WithResponseHeader("X-Group", "yes"),
	)
	shiftapi.Get(g, "/item", func(r *http.Request, in struct{}) (Greeting, error) {
		return Greeting{Hello: "ok"}, nil
	}, shiftapi.WithResponseHeader("X-Route", "yes"))

	resp := doRequest(t, api, "GET", "/v1/item", "")
	for _, name := range []string{"X-Global", "X-Group", "X-Route"} {
		if got := resp.Header.Get(name); got != "yes" {
			t.Errorf("expected %s %q, got %q", name, "yes", got)
		}
	}
}

func TestWithResponseHeaderCombinedWithStructTag(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/combo", func(r *http.Request, in struct{}) (RespWithHeader, error) {
		return RespWithHeader{CacheControl: "private", Message: "ok"}, nil
	}, shiftapi.WithResponseHeader("X-Extra", "value"))

	resp := doRequest(t, api, "GET", "/combo", "")
	if got := resp.Header.Get("Cache-Control"); got != "private" {
		t.Errorf("expected Cache-Control %q, got %q", "private", got)
	}
	if got := resp.Header.Get("X-Extra"); got != "value" {
		t.Errorf("expected X-Extra %q, got %q", "value", got)
	}
}

func TestWithResponseHeaderOpenAPISchema(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/secure", func(r *http.Request, in struct{}) (Greeting, error) {
		return Greeting{}, nil
	}, shiftapi.WithResponseHeader("X-Content-Type-Options", "nosniff"))

	spec := api.Spec()
	pathItem := spec.Paths.Find("/secure")
	if pathItem == nil || pathItem.Get == nil {
		t.Fatal("expected GET /secure in spec")
	}
	respRef := pathItem.Get.Responses.Value("200")
	if respRef == nil || respRef.Value == nil {
		t.Fatal("expected 200 response in spec")
	}
	if respRef.Value.Headers == nil {
		t.Fatal("expected response headers in spec")
	}
	hdr := respRef.Value.Headers["X-Content-Type-Options"]
	if hdr == nil || hdr.Value == nil {
		t.Fatal("expected X-Content-Type-Options response header in spec")
	}
	if !hdr.Value.Required {
		t.Error("expected static response header to be required")
	}
	if !hdr.Value.Schema.Value.Type.Is("string") {
		t.Error("expected string type for static response header")
	}
}

func TestWithResponseHeaderNestedGroups(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithResponseHeader("X-Level", "api"),
	)
	g1 := api.Group("/g1",
		shiftapi.WithResponseHeader("X-Level", "group"),
	)
	g2 := g1.Group("/g2")
	shiftapi.Get(g2, "/item", func(r *http.Request, in struct{}) (Greeting, error) {
		return Greeting{Hello: "ok"}, nil
	})

	resp := doRequest(t, api, "GET", "/g1/g2/item", "")
	// Last set wins for same header name (group overrides api).
	if got := resp.Header.Get("X-Level"); got != "group" {
		t.Errorf("expected X-Level %q, got %q", "group", got)
	}
}

// --- No-body response tests ---

func TestNoBody_StructEmpty_204(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Delete(api, "/items/{id}", func(r *http.Request, _ struct{}) (struct{}, error) {
		return struct{}{}, nil
	}, shiftapi.WithStatus(http.StatusNoContent))

	resp := doRequest(t, api, "DELETE", "/items/1", "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 0 {
		t.Errorf("expected empty body, got %q", body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		t.Errorf("expected no Content-Type, got %q", ct)
	}
}

func TestNoBody_PointerStructEmpty_204(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Delete(api, "/items/{id}", func(r *http.Request, _ struct{}) (*Empty, error) {
		return &Empty{}, nil
	}, shiftapi.WithStatus(http.StatusNoContent))

	resp := doRequest(t, api, "DELETE", "/items/1", "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 0 {
		t.Errorf("expected empty body, got %q", body)
	}
}

func TestNoBody_HeaderOnly_204(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Delete(api, "/items/{id}", func(r *http.Request, _ struct{}) (RespHeaderOnly, error) {
		return RespHeaderOnly{Location: "/items"}, nil
	}, shiftapi.WithStatus(http.StatusNoContent))

	resp := doRequest(t, api, "DELETE", "/items/1", "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/items" {
		t.Errorf("expected Location header %q, got %q", "/items", got)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 0 {
		t.Errorf("expected empty body, got %q", body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		t.Errorf("expected no Content-Type, got %q", ct)
	}
}

func TestNoBody_304_NotModified(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/item", func(r *http.Request, _ struct{}) (struct{}, error) {
		return struct{}{}, nil
	}, shiftapi.WithStatus(http.StatusNotModified))

	resp := doRequest(t, api, "GET", "/item", "")
	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 0 {
		t.Errorf("expected empty body, got %q", body)
	}
}

func TestNoBody_WithStaticHeaders_204(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Delete(api, "/items/{id}", func(r *http.Request, _ struct{}) (struct{}, error) {
		return struct{}{}, nil
	}, shiftapi.WithStatus(http.StatusNoContent),
		shiftapi.WithResponseHeader("X-Deleted", "true"),
	)

	resp := doRequest(t, api, "DELETE", "/items/1", "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("X-Deleted"); got != "true" {
		t.Errorf("expected X-Deleted header %q, got %q", "true", got)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 0 {
		t.Errorf("expected empty body, got %q", body)
	}
}

func TestNoBody_OpenAPI_204_NoContent(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Delete(api, "/items/{id}", func(r *http.Request, _ struct{}) (struct{}, error) {
		return struct{}{}, nil
	}, shiftapi.WithStatus(http.StatusNoContent))

	spec := api.Spec()
	respRef := spec.Paths.Find("/items/{id}").Delete.Responses.Value("204")
	if respRef == nil || respRef.Value == nil {
		t.Fatal("expected 204 response in spec")
	}
	if respRef.Value.Content != nil {
		t.Error("expected no content in 204 response spec")
	}
	if *respRef.Value.Description != "No Content" {
		t.Errorf("expected description %q, got %q", "No Content", *respRef.Value.Description)
	}
}

func TestNoBody_OpenAPI_204_WithHeaders(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Delete(api, "/items/{id}", func(r *http.Request, _ struct{}) (RespHeaderOnly, error) {
		return RespHeaderOnly{Location: "/items"}, nil
	}, shiftapi.WithStatus(http.StatusNoContent))

	spec := api.Spec()
	respRef := spec.Paths.Find("/items/{id}").Delete.Responses.Value("204")
	if respRef == nil || respRef.Value == nil {
		t.Fatal("expected 204 response in spec")
	}
	if respRef.Value.Content != nil {
		t.Error("expected no content in 204 response spec")
	}
	if respRef.Value.Headers == nil {
		t.Fatal("expected headers in 204 response spec")
	}
	if respRef.Value.Headers["Location"] == nil {
		t.Error("expected Location header in 204 response spec")
	}
}

func TestNoBody_PanicsOnBodyWith204(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for 204 with body fields")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T", r)
		}
		if !strings.Contains(msg, "204") || !strings.Contains(msg, "must not have a response body") {
			t.Errorf("unexpected panic message: %s", msg)
		}
	}()

	api := newTestAPI(t)
	shiftapi.Delete(api, "/items/{id}", func(r *http.Request, _ struct{}) (Greeting, error) {
		return Greeting{}, nil
	}, shiftapi.WithStatus(http.StatusNoContent))
}

// --- Slice/array response tests ---

type SliceItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestSliceResponse_Runtime(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/items", func(r *http.Request, _ struct{}) ([]SliceItem, error) {
		return []SliceItem{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}, nil
	})

	resp := doRequest(t, api, "GET", "/items", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var items []SliceItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ID != 1 || items[1].Name != "b" {
		t.Errorf("unexpected items: %+v", items)
	}
}

func TestSliceResponse_NilSlice(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/items", func(r *http.Request, _ struct{}) ([]SliceItem, error) {
		return nil, nil
	})

	resp := doRequest(t, api, "GET", "/items", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	// encoding/json encodes nil slices as "null"
	got := strings.TrimSpace(string(body))
	if got != "null" {
		t.Errorf("expected null, got %q", got)
	}
}

func TestSliceResponse_EmptySlice(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/items", func(r *http.Request, _ struct{}) ([]SliceItem, error) {
		return []SliceItem{}, nil
	})

	resp := doRequest(t, api, "GET", "/items", "")
	body, _ := io.ReadAll(resp.Body)
	got := strings.TrimSpace(string(body))
	if got != "[]" {
		t.Errorf("expected [], got %q", got)
	}
}

func TestSliceResponse_StringSlice(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/tags", func(r *http.Request, _ struct{}) ([]string, error) {
		return []string{"go", "api"}, nil
	})

	resp := doRequest(t, api, "GET", "/tags", "")
	var tags []string
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(tags) != 2 || tags[0] != "go" {
		t.Errorf("unexpected tags: %v", tags)
	}
}

func TestSliceResponse_OpenAPI_ArraySchema(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/items", func(r *http.Request, _ struct{}) ([]SliceItem, error) {
		return nil, nil
	})

	spec := api.Spec()
	respRef := spec.Paths.Find("/items").Get.Responses.Value("200")
	if respRef == nil || respRef.Value == nil {
		t.Fatal("expected 200 response in spec")
	}
	ct := respRef.Value.Content["application/json"]
	if ct == nil || ct.Schema == nil {
		t.Fatal("expected application/json content with schema")
	}
	if ct.Schema.Value == nil || !ct.Schema.Value.Type.Is("array") {
		t.Fatal("expected array type in response schema")
	}
	items := ct.Schema.Value.Items
	if items == nil {
		t.Fatal("expected items in array schema")
	}
	if items.Ref != "#/components/schemas/SliceItem" {
		t.Errorf("expected $ref to SliceItem, got %q", items.Ref)
	}
	// Verify SliceItem is in components
	si := spec.Components.Schemas["SliceItem"]
	if si == nil || si.Value == nil {
		t.Fatal("expected SliceItem in components/schemas")
	}
	if si.Value.Properties["id"] == nil || si.Value.Properties["name"] == nil {
		t.Error("expected id and name properties on SliceItem schema")
	}
}

func TestSliceResponse_OpenAPI_StringSlice(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/tags", func(r *http.Request, _ struct{}) ([]string, error) {
		return nil, nil
	})

	spec := api.Spec()
	respRef := spec.Paths.Find("/tags").Get.Responses.Value("200")
	ct := respRef.Value.Content["application/json"]
	if ct == nil || ct.Schema == nil || ct.Schema.Value == nil {
		t.Fatal("expected schema")
	}
	if !ct.Schema.Value.Type.Is("array") {
		t.Fatal("expected array type")
	}
	if ct.Schema.Value.Items == nil || !ct.Schema.Value.Items.Value.Type.Is("string") {
		t.Error("expected string items type")
	}
}
