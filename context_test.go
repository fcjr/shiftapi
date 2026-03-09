package shiftapi_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fcjr/shiftapi"
)

func TestFromContext_roundTrip(t *testing.T) {
	key := shiftapi.NewContextKey[string]("user")
	r := httptest.NewRequest("GET", "/", nil)
	r = shiftapi.SetContext(r, key, "alice")

	got, ok := shiftapi.FromContext(r, key)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "alice" {
		t.Fatalf("got %q, want %q", got, "alice")
	}
}

func TestFromContext_missing(t *testing.T) {
	key := shiftapi.NewContextKey[string]("user")
	r := httptest.NewRequest("GET", "/", nil)

	got, ok := shiftapi.FromContext(r, key)
	if ok {
		t.Fatal("expected ok=false for missing key")
	}
	if got != "" {
		t.Fatalf("expected zero value, got %q", got)
	}
}

func TestFromContext_distinctKeysSameType(t *testing.T) {
	key1 := shiftapi.NewContextKey[string]("first")
	key2 := shiftapi.NewContextKey[string]("second")

	r := httptest.NewRequest("GET", "/", nil)
	r = shiftapi.SetContext(r, key1, "one")
	r = shiftapi.SetContext(r, key2, "two")

	got1, ok1 := shiftapi.FromContext(r, key1)
	got2, ok2 := shiftapi.FromContext(r, key2)

	if !ok1 || got1 != "one" {
		t.Fatalf("key1: got %q ok=%v, want %q ok=true", got1, ok1, "one")
	}
	if !ok2 || got2 != "two" {
		t.Fatalf("key2: got %q ok=%v, want %q ok=true", got2, ok2, "two")
	}
}

func TestFromContext_structValue(t *testing.T) {
	type User struct {
		ID   int
		Name string
	}

	key := shiftapi.NewContextKey[User]("user")
	r := httptest.NewRequest("GET", "/", nil)
	r = shiftapi.SetContext(r, key, User{ID: 42, Name: "alice"})

	got, ok := shiftapi.FromContext(r, key)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got.ID != 42 || got.Name != "alice" {
		t.Fatalf("got %+v, want {ID:42 Name:alice}", got)
	}
}

func TestFromContext_middlewareIntegration(t *testing.T) {
	userKey := shiftapi.NewContextKey[string]("user")

	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, shiftapi.SetContext(r, userKey, "alice"))
		})
	}

	api := shiftapi.New(shiftapi.WithMiddleware(authMiddleware))

	type Resp struct {
		User string `json:"user"`
	}

	shiftapi.Handle(api, "GET /whoami", func(r *http.Request, _ struct{}) (*Resp, error) {
		user, ok := shiftapi.FromContext(r, userKey)
		if !ok {
			return nil, fmt.Errorf("no user in context")
		}
		return &Resp{User: user}, nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/whoami", nil)
	api.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}
	if want := `{"user":"alice"}` + "\n"; w.Body.String() != want {
		t.Fatalf("body=%q, want %q", w.Body.String(), want)
	}
}

func TestContextKey_String(t *testing.T) {
	key := shiftapi.NewContextKey[int]("request-id")
	if got := key.String(); got != "request-id" {
		t.Fatalf("got %q, want %q", got, "request-id")
	}
}
