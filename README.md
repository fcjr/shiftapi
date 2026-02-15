
<p align="center">
	<img src="assets/logo.svg" alt="ShiftAPI Logo">
</p>

# ShiftAPI

Quickly write RESTful APIs in Go with automatic OpenAPI 3.1 schema generation.

Inspired by the simplicity of [FastAPI](https://github.com/tiangolo/fastapi).

<!-- [![GitHub release (latest by date)][release-img]][release] -->
[![GolangCI][golangci-lint-img]][golangci-lint]
[![Go Report Card][report-card-img]][report-card]

## Installation

```sh
go get github.com/fcjr/shiftapi
```

## Quick Start

```go
package main

import (
    "log"
    "net/http"

    "github.com/fcjr/shiftapi"
)

type Person struct {
    Name string `json:"name"`
}

type Greeting struct {
    Hello string `json:"hello"`
}

func greet(r *http.Request, body *Person) (*Greeting, error) {
    if body.Name == "" {
        return nil, shiftapi.Error(http.StatusBadRequest, "name is required")
    }
    return &Greeting{Hello: body.Name}, nil
}

func main() {
    api := shiftapi.New(shiftapi.WithInfo(shiftapi.Info{
        Title:       "Greeter API",
        Description: "It greets you by name.",
        Version:     "1.0.0",
    }))

    shiftapi.Post(api, "/greet", greet)

    log.Println("listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", api))
    // docs at http://localhost:8080/docs
}
```

## How It Works

ShiftAPI is a thin layer on top of `net/http`. The `API` type implements `http.Handler`, so it works with the standard library server, middleware, and testing tools.

Generic free functions (`Get`, `Post`, `Put`, etc.) capture your request/response types at compile time for two purposes:

1. **Type-safe request handling** — request bodies are automatically decoded from JSON and passed to your handler as typed values.
2. **Automatic OpenAPI generation** — the types are reflected into an OpenAPI 3.1 spec served at `/openapi.json`, with interactive docs at `/docs`.

## Usage

### Handlers with a request body (POST, PUT, PATCH)

```go
shiftapi.Post(api, "/users", func(r *http.Request, body *CreateUser) (*User, error) {
    user, err := db.CreateUser(r.Context(), body)
    if err != nil {
        return nil, err
    }
    return user, nil
}, shiftapi.WithStatus(http.StatusCreated))
```

### Handlers without a request body (GET, DELETE, HEAD)

```go
shiftapi.Get(api, "/users/{id}", func(r *http.Request) (*User, error) {
    id := r.PathValue("id")  // standard Go 1.22+ path params
    return db.GetUser(r.Context(), id)
})
```

Since the handler receives a standard `*http.Request`, you have full access to path params, query params, headers, cookies, context — everything you'd have in a regular `http.HandlerFunc`.

### Error Handling

Return `shiftapi.Error` to control the HTTP status code and message:

```go
shiftapi.Get(api, "/users/{id}", func(r *http.Request) (*User, error) {
    user, err := db.GetUser(r.Context(), r.PathValue("id"))
    if err != nil {
        return nil, shiftapi.Error(http.StatusNotFound, "user not found")
    }
    return user, nil
})
```

Any non-`APIError` returns a `500 Internal Server Error`. `APIError` responses are returned as JSON:

```json
{"message": "user not found"}
```

### Route Metadata

Add OpenAPI metadata with `WithRouteInfo`:

```go
shiftapi.Post(api, "/greet", greet,
    shiftapi.WithRouteInfo(shiftapi.RouteInfo{
        Summary:     "Greet a person",
        Description: "Greet a person with a friendly greeting",
        Tags:        []string{"greetings"},
    }),
)
```

### Middleware

Since `API` implements `http.Handler`, any standard middleware works:

```go
api := shiftapi.New()
shiftapi.Get(api, "/health", healthHandler)

wrapped := loggingMiddleware(corsMiddleware(api))
http.ListenAndServe(":8080", wrapped)
```

### Mounting Under a Prefix

```go
api := shiftapi.New()
shiftapi.Get(api, "/health", healthHandler)

mux := http.NewServeMux()
mux.Handle("/api/v1/", http.StripPrefix("/api/v1", api))
http.ListenAndServe(":8080", mux)
```

### Testing

Use `httptest` directly:

```go
func TestHealthEndpoint(t *testing.T) {
    api := shiftapi.New()
    shiftapi.Get(api, "/health", healthHandler)

    req := httptest.NewRequest(http.MethodGet, "/health", nil)
    rec := httptest.NewRecorder()
    api.ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }
}
```

[release-img]: https://img.shields.io/github/v/release/fcjr/shiftapi
[release]: https://github.com/fcjr/shiftapi/releases
[golangci-lint-img]: https://github.com/fcjr/shiftapi/workflows/go-lint/badge.svg
[golangci-lint]: https://github.com/fcjr/shiftapi/actions?query=workflow%3Ago-lint
[report-card-img]: https://goreportcard.com/badge/github.com/fcjr/shiftapi
[report-card]: https://goreportcard.com/report/github.com/fcjr/shiftapi
