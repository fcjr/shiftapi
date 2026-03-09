package shiftapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fcjr/shiftapi"
)

type TaskStatus string

const (
	TaskStatusActive   TaskStatus = "active"
	TaskStatusInactive TaskStatus = "inactive"
	TaskStatusPending  TaskStatus = "pending"
)

type TaskPriority int

const (
	TaskPriorityLow    TaskPriority = 1
	TaskPriorityMedium TaskPriority = 2
	TaskPriorityHigh   TaskPriority = 3
)

func TestWithEnum_bodyField(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithEnum[TaskStatus](TaskStatusActive, TaskStatusInactive, TaskStatusPending),
	)

	type Req struct {
		Status TaskStatus `json:"status"`
	}
	type Resp struct {
		OK bool `json:"ok"`
	}

	shiftapi.Handle(api, "POST /items", func(r *http.Request, in *Req) (*Resp, error) {
		return &Resp{OK: true}, nil
	})

	spec := fetchSpec(t, api)
	schema := componentSchema(t, spec, "Req")
	statusProp := schema["properties"].(map[string]any)["status"].(map[string]any)
	enumVals := statusProp["enum"].([]any)

	assertEnumValues(t, enumVals, []string{"active", "inactive", "pending"})
}

func TestWithEnum_queryParam(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithEnum[TaskStatus](TaskStatusActive, TaskStatusInactive, TaskStatusPending),
	)

	type Req struct {
		Filter TaskStatus `query:"filter"`
	}
	type Resp struct {
		OK bool `json:"ok"`
	}

	shiftapi.Handle(api, "GET /items", func(r *http.Request, in *Req) (*Resp, error) {
		return &Resp{OK: true}, nil
	})

	spec := fetchSpec(t, api)
	params := operationParams(t, spec, "/items", "get")

	var found bool
	for _, p := range params {
		pm := p.(map[string]any)
		if pm["name"] == "filter" {
			schema := pm["schema"].(map[string]any)
			enumVals := schema["enum"].([]any)
			assertEnumValues(t, enumVals, []string{"active", "inactive", "pending"})
			found = true
		}
	}
	if !found {
		t.Fatal("query param 'filter' not found in spec")
	}
}

func TestWithEnum_pathParam(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithEnum[TaskStatus](TaskStatusActive, TaskStatusInactive, TaskStatusPending),
	)

	type Req struct {
		Status TaskStatus `path:"status"`
	}
	type Resp struct {
		OK bool `json:"ok"`
	}

	shiftapi.Handle(api, "GET /items/{status}", func(r *http.Request, in *Req) (*Resp, error) {
		return &Resp{OK: true}, nil
	})

	spec := fetchSpec(t, api)
	params := operationParams(t, spec, "/items/{status}", "get")

	var found bool
	for _, p := range params {
		pm := p.(map[string]any)
		if pm["name"] == "status" {
			schema := pm["schema"].(map[string]any)
			enumVals := schema["enum"].([]any)
			assertEnumValues(t, enumVals, []string{"active", "inactive", "pending"})
			found = true
		}
	}
	if !found {
		t.Fatal("path param 'status' not found in spec")
	}
}

func TestWithEnum_headerParam(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithEnum[TaskStatus](TaskStatusActive, TaskStatusInactive, TaskStatusPending),
	)

	type Req struct {
		Status TaskStatus `header:"X-Status"`
	}
	type Resp struct {
		OK bool `json:"ok"`
	}

	shiftapi.Handle(api, "GET /items", func(r *http.Request, in *Req) (*Resp, error) {
		return &Resp{OK: true}, nil
	})

	spec := fetchSpec(t, api)
	params := operationParams(t, spec, "/items", "get")

	var found bool
	for _, p := range params {
		pm := p.(map[string]any)
		if pm["name"] == "X-Status" {
			schema := pm["schema"].(map[string]any)
			enumVals := schema["enum"].([]any)
			assertEnumValues(t, enumVals, []string{"active", "inactive", "pending"})
			found = true
		}
	}
	if !found {
		t.Fatal("header param 'X-Status' not found in spec")
	}
}

func TestWithEnum_intEnum(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithEnum[TaskPriority](TaskPriorityLow, TaskPriorityMedium, TaskPriorityHigh),
	)

	type Req struct {
		Priority TaskPriority `json:"priority"`
	}
	type Resp struct {
		OK bool `json:"ok"`
	}

	shiftapi.Handle(api, "POST /tasks", func(r *http.Request, in *Req) (*Resp, error) {
		return &Resp{OK: true}, nil
	})

	spec := fetchSpec(t, api)
	schema := componentSchema(t, spec, "Req")
	priorityProp := schema["properties"].(map[string]any)["priority"].(map[string]any)
	enumVals := priorityProp["enum"].([]any)

	// JSON numbers are float64
	want := []float64{1, 2, 3}
	if len(enumVals) != len(want) {
		t.Fatalf("enum length = %d, want %d", len(enumVals), len(want))
	}
	for i, v := range enumVals {
		got, ok := v.(float64)
		if !ok {
			t.Fatalf("enum[%d] is %T, want float64", i, v)
		}
		if got != want[i] {
			t.Fatalf("enum[%d] = %v, want %v", i, got, want[i])
		}
	}
}

func TestWithEnum_oneofOverridesEnum(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithEnum[TaskStatus](TaskStatusActive, TaskStatusInactive, TaskStatusPending),
	)

	type Req struct {
		// oneof tag should override WithEnum
		Status TaskStatus `json:"status" validate:"oneof=active inactive"`
	}
	type Resp struct {
		OK bool `json:"ok"`
	}

	shiftapi.Handle(api, "POST /items", func(r *http.Request, in *Req) (*Resp, error) {
		return &Resp{OK: true}, nil
	})

	spec := fetchSpec(t, api)
	schema := componentSchema(t, spec, "Req")
	statusProp := schema["properties"].(map[string]any)["status"].(map[string]any)
	enumVals := statusProp["enum"].([]any)

	// Should only have the oneof values, not the registered enum values
	assertEnumValues(t, enumVals, []string{"active", "inactive"})
}

func TestWithEnum_pointerField(t *testing.T) {
	api := shiftapi.New(
		shiftapi.WithEnum[TaskStatus](TaskStatusActive, TaskStatusInactive, TaskStatusPending),
	)

	type Req struct {
		Status *TaskStatus `json:"status,omitempty"`
	}
	type Resp struct {
		OK bool `json:"ok"`
	}

	shiftapi.Handle(api, "POST /items", func(r *http.Request, in *Req) (*Resp, error) {
		return &Resp{OK: true}, nil
	})

	spec := fetchSpec(t, api)
	schema := componentSchema(t, spec, "Req")
	statusProp := schema["properties"].(map[string]any)["status"].(map[string]any)
	enumVals := statusProp["enum"].([]any)

	assertEnumValues(t, enumVals, []string{"active", "inactive", "pending"})
}

// --- helpers ---

func fetchSpec(t *testing.T, api *shiftapi.API) map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/openapi.json", nil)
	api.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /openapi.json status=%d", w.Code)
	}
	var spec map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Fatalf("unmarshal spec: %v", err)
	}
	return spec
}

func componentSchema(t *testing.T, spec map[string]any, name string) map[string]any {
	t.Helper()
	components := spec["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	schema, ok := schemas[name].(map[string]any)
	if !ok {
		t.Fatalf("component schema %q not found", name)
	}
	return schema
}

func operationParams(t *testing.T, spec map[string]any, path, method string) []any {
	t.Helper()
	paths := spec["paths"].(map[string]any)
	pathItem := paths[path].(map[string]any)
	op := pathItem[method].(map[string]any)
	params, ok := op["parameters"].([]any)
	if !ok {
		t.Fatal("no parameters found")
	}
	return params
}

func assertEnumValues(t *testing.T, got []any, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("enum length = %d, want %d: %v", len(got), len(want), got)
	}
	for i, v := range got {
		s, ok := v.(string)
		if !ok {
			t.Fatalf("enum[%d] is %T, want string", i, v)
		}
		if s != want[i] {
			t.Fatalf("enum[%d] = %q, want %q", i, s, want[i])
		}
	}
}
