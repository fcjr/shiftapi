package shiftapi_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
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
	shiftapi.Get(api, "/health", func(r *http.Request) (*Status, error) {
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
	shiftapi.Post(api, "/greet", func(r *http.Request, body *Person) (*Greeting, error) {
		return &Greeting{Hello: body.Name}, nil
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
	shiftapi.Post(api, "/greet", func(r *http.Request, body *Person) (*Greeting, error) {
		return &Greeting{Hello: body.Name}, nil
	})

	resp := doRequest(t, api, http.MethodPost, "/greet", `not json`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPostHandlerEmptyBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/greet", func(r *http.Request, body *Person) (*Greeting, error) {
		return &Greeting{Hello: body.Name}, nil
	})

	resp := doRequest(t, api, http.MethodPost, "/greet", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPostHandlerEmptyJSONObject(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/person", func(r *http.Request, body *ValidatedPerson) (*ValidatedPerson, error) {
		return body, nil
	})

	resp := doRequest(t, api, http.MethodPost, "/person", `{}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

// --- GET handler tests ---

func TestGetHandler(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/health", func(r *http.Request) (*Status, error) {
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
	shiftapi.Get(api, "/items/{id}", func(r *http.Request) (*Item, error) {
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
	shiftapi.Put(api, "/items/{id}", func(r *http.Request, body *Item) (*Item, error) {
		body.ID = r.PathValue("id")
		return body, nil
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
	shiftapi.Patch(api, "/items/{id}", func(r *http.Request, body *Item) (*Item, error) {
		body.ID = r.PathValue("id")
		return body, nil
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
	shiftapi.Delete(api, "/items/{id}", func(r *http.Request) (*Empty, error) {
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
	shiftapi.Head(api, "/ping", func(r *http.Request) (*Empty, error) {
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
	shiftapi.Options(api, "/items", func(r *http.Request) (*Empty, error) {
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
	shiftapi.Trace(api, "/debug", func(r *http.Request) (*Empty, error) {
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
	shiftapi.Connect(api, "/tunnel", func(r *http.Request) (*Empty, error) {
		return &Empty{}, nil
	})

	resp := doRequest(t, api, http.MethodConnect, "/tunnel", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// --- Error handling tests ---

func TestAPIErrorReturnsCorrectStatusCode(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/fail", func(r *http.Request) (*Empty, error) {
		return nil, shiftapi.Error(http.StatusNotFound, "not found")
	})

	resp := doRequest(t, api, http.MethodGet, "/fail", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	body := decodeJSON[map[string]string](t, resp)
	if body["message"] != "not found" {
		t.Errorf("expected message 'not found', got %q", body["message"])
	}
}

func TestAPIErrorReturnsJSON(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/fail", func(r *http.Request, body *Person) (*Greeting, error) {
		return nil, shiftapi.Error(http.StatusUnprocessableEntity, "invalid data")
	})

	resp := doRequest(t, api, http.MethodPost, "/fail", `{"name":"test"}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type application/json; charset=utf-8, got %q", ct)
	}
}

func TestGenericErrorReturns500(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/boom", func(r *http.Request) (*Empty, error) {
		return nil, errors.New("something broke")
	})

	resp := doRequest(t, api, http.MethodGet, "/boom", "")
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

func TestAPIErrorSatisfiesErrorsAs(t *testing.T) {
	err := shiftapi.Error(http.StatusBadRequest, "bad")
	var apiErr *shiftapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected errors.As to match *APIError")
	}
	if apiErr.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.Status)
	}
}

func TestAPIErrorMessage(t *testing.T) {
	err := shiftapi.Error(http.StatusForbidden, "forbidden")
	expected := "403: forbidden"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

// --- WithStatus tests ---

func TestWithStatusCustomCode(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/items", func(r *http.Request, body *Item) (*Item, error) {
		body.ID = "new-id"
		return body, nil
	}, shiftapi.WithStatus(http.StatusCreated))

	resp := doRequest(t, api, http.MethodPost, "/items", `{"name":"widget"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestWithStatusOnGetHandler(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Delete(api, "/items/{id}", func(r *http.Request) (*Empty, error) {
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
	shiftapi.Post(api, "/greet", func(r *http.Request, body *Person) (*Greeting, error) {
		return &Greeting{Hello: body.Name}, nil
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
	shiftapi.Get(api, "/health", func(r *http.Request) (*Status, error) {
		return &Status{OK: true}, nil
	})

	spec := api.Spec()
	if spec.Paths.Find("/health") == nil {
		t.Fatal("expected /health in spec paths")
	}
}

func TestSpecGetHasNoRequestBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/health", func(r *http.Request) (*Status, error) {
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
	shiftapi.Post(api, "/greet", func(r *http.Request, body *Person) (*Greeting, error) {
		return &Greeting{Hello: body.Name}, nil
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
	shiftapi.Post(api, "/greet", func(r *http.Request, body *Person) (*Greeting, error) {
		return &Greeting{Hello: body.Name}, nil
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

func TestSpecDeleteHasNoRequestBody(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Delete(api, "/items/{id}", func(r *http.Request) (*Empty, error) {
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
	shiftapi.Get(api, "/health", func(r *http.Request) (*Status, error) {
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
	shiftapi.Get(api, "/health", func(r *http.Request) (*Status, error) {
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
	shiftapi.Post(api, "/items", func(r *http.Request, body *Item) (*Item, error) {
		return body, nil
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
	shiftapi.Post(api, "/greet", func(r *http.Request, body *Person) (*Greeting, error) {
		return &Greeting{Hello: body.Name}, nil
	})

	spec := api.Spec()
	if len(spec.Components.Schemas) == 0 {
		t.Fatal("expected component schemas to be populated")
	}
}

func TestSpecMultipleMethodsOnSamePath(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/items", func(r *http.Request) (*[]Item, error) {
		return &[]Item{}, nil
	})
	shiftapi.Post(api, "/items", func(r *http.Request, body *Item) (*Item, error) {
		return body, nil
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
	shiftapi.Get(api, "/users/{id}", func(r *http.Request) (*Item, error) {
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
	shiftapi.Get(api, "/orgs/{orgId}/users/{userId}", func(r *http.Request) (*Item, error) {
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
	shiftapi.Get(api, "/health", func(r *http.Request) (*Status, error) {
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
				shiftapi.Get(api, tc.path, func(r *http.Request) (*Empty, error) {
					return &Empty{}, nil
				})
			case "POST":
				shiftapi.Post(api, tc.path, func(r *http.Request, body *Empty) (*Empty, error) {
					return &Empty{}, nil
				})
			case "PUT":
				shiftapi.Put(api, tc.path, func(r *http.Request, body *Empty) (*Empty, error) {
					return &Empty{}, nil
				})
			case "DELETE":
				shiftapi.Delete(api, tc.path, func(r *http.Request) (*Empty, error) {
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

func TestSpecHasDefaultErrorResponse(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/health", func(r *http.Request) (*Status, error) {
		return &Status{OK: true}, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/health").Get
	defaultResp := op.Responses.Value("default")
	if defaultResp == nil {
		t.Fatal("expected default error response in spec")
	}
	if defaultResp.Value.Description == nil || *defaultResp.Value.Description != "Error" {
		t.Error("expected default response description 'Error'")
	}
	content := defaultResp.Value.Content["application/json"]
	if content == nil {
		t.Fatal("expected application/json content in default error response")
	}
	msgProp := content.Schema.Value.Properties["message"]
	if msgProp == nil {
		t.Fatal("expected 'message' property in error schema")
	}
}

func TestSpecDefaultErrorResponseOnPost(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/items", func(r *http.Request, body *Item) (*Item, error) {
		return body, nil
	})

	spec := api.Spec()
	op := spec.Paths.Find("/items").Post
	if op.Responses.Value("default") == nil {
		t.Fatal("expected default error response on POST")
	}
}

// --- HTTP Handler interface tests ---

func TestAPIImplementsHTTPHandler(t *testing.T) {
	var _ http.Handler = shiftapi.New()
}

func TestHTTPTestServerCompatibility(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/ping", func(r *http.Request) (*Status, error) {
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
	shiftapi.Get(api, "/health", func(r *http.Request) (*Status, error) {
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
	shiftapi.Get(api, "/health", func(r *http.Request) (*Status, error) {
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
	shiftapi.Get(api, "/echo-header", func(r *http.Request) (*map[string]string, error) {
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
	shiftapi.Get(api, "/search", func(r *http.Request) (*map[string]string, error) {
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
	shiftapi.Get(api, "/ctx", func(r *http.Request) (*Status, error) {
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
	shiftapi.Get(api, "/test", func(r *http.Request) (*Status, error) {
		return &Status{OK: true}, nil
	})

	resp := doRequest(t, api, http.MethodGet, "/test", "")
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type application/json; charset=utf-8, got %q", ct)
	}
}

func TestErrorResponseFromAPIErrorHasJSONContentType(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Get(api, "/fail", func(r *http.Request) (*Empty, error) {
		return nil, shiftapi.Error(http.StatusBadRequest, "bad")
	})

	resp := doRequest(t, api, http.MethodGet, "/fail", "")
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type application/json; charset=utf-8, got %q", ct)
	}
}

// --- Multiple routes tests ---

func TestMultipleRoutes(t *testing.T) {
	api := newTestAPI(t)

	shiftapi.Get(api, "/a", func(r *http.Request) (*map[string]string, error) {
		return &map[string]string{"route": "a"}, nil
	})
	shiftapi.Get(api, "/b", func(r *http.Request) (*map[string]string, error) {
		return &map[string]string{"route": "b"}, nil
	})
	shiftapi.Post(api, "/c", func(r *http.Request, body *Empty) (*map[string]string, error) {
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

	shiftapi.Get(api, "/new-route", func(r *http.Request) (*Empty, error) {
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
	shiftapi.Post(api, "/person", func(r *http.Request, body *ValidatedPerson) (*ValidatedPerson, error) {
		return body, nil
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
	shiftapi.Post(api, "/person", func(r *http.Request, body *ValidatedPerson) (*ValidatedPerson, error) {
		return body, nil
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
	shiftapi.Post(api, "/minmax", func(r *http.Request, body *MinMaxBody) (*MinMaxBody, error) {
		return body, nil
	})

	resp := doRequest(t, api, http.MethodPost, "/minmax", `{"age":0,"name":"a"}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestValidationValidPayloadPassesThrough(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/person", func(r *http.Request, body *ValidatedPerson) (*ValidatedPerson, error) {
		return body, nil
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
	shiftapi.Post(api, "/noval", func(r *http.Request, body *NoValidateBody) (*NoValidateBody, error) {
		return body, nil
	})

	resp := doRequest(t, api, http.MethodPost, "/noval", `{"foo":"bar"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestWithValidatorCustomInstance(t *testing.T) {
	v := validator.New()
	api := shiftapi.New(shiftapi.WithValidator(v))
	shiftapi.Post(api, "/person", func(r *http.Request, body *ValidatedPerson) (*ValidatedPerson, error) {
		return body, nil
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
	var valErr *shiftapi.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatal("expected errors.As to match *ValidationError")
	}
	if valErr.Message != "validation failed" {
		t.Errorf("expected message 'validation failed', got %q", valErr.Message)
	}
}

// --- Validation spec tests ---

func TestSpecRequiredFieldInParentSchema(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/person", func(r *http.Request, body *ValidatedPerson) (*ValidatedPerson, error) {
		return body, nil
	})

	spec := api.Spec()
	// Find the ValidatedPerson schema in components
	schemaRef, ok := spec.Components.Schemas["ValidatedPerson"]
	if !ok {
		t.Fatal("expected ValidatedPerson in component schemas")
	}
	schema := schemaRef.Value
	if !contains(schema.Required, "name") {
		t.Errorf("expected 'name' in required, got %v", schema.Required)
	}
	if !contains(schema.Required, "email") {
		t.Errorf("expected 'email' in required, got %v", schema.Required)
	}
}

func TestSpecEmailFormatSet(t *testing.T) {
	api := newTestAPI(t)
	shiftapi.Post(api, "/person", func(r *http.Request, body *ValidatedPerson) (*ValidatedPerson, error) {
		return body, nil
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
	shiftapi.Post(api, "/minmax", func(r *http.Request, body *MinMaxBody) (*MinMaxBody, error) {
		return body, nil
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
	shiftapi.Post(api, "/minmax", func(r *http.Request, body *MinMaxBody) (*MinMaxBody, error) {
		return body, nil
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
	shiftapi.Post(api, "/oneof", func(r *http.Request, body *OneOfBody) (*OneOfBody, error) {
		return body, nil
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

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
