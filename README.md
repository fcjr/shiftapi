
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
    Name string `json:"name" validate:"required"`
}

type Greeting struct {
    Hello string `json:"hello"`
}

func greet(r *http.Request, body *Person) (*Greeting, error) {
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
    log.Fatal(shiftapi.ListenAndServe(":8080", api))
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

### Validation

ShiftAPI has built-in validation powered by [go-playground/validator](https://github.com/go-playground/validator). Add `validate` struct tags to your request types — they are enforced at runtime and automatically reflected in the OpenAPI schema.

```go
type CreateUser struct {
    Name  string `json:"name"  validate:"required,min=2,max=50"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age"   validate:"gte=0,lte=150"`
    Role  string `json:"role"  validate:"oneof=admin user guest"`
}
```

Invalid requests return a `422 Unprocessable Entity` with per-field errors:

```json
{
    "message": "validation failed",
    "errors": [
        { "field": "Name",  "message": "this field is required" },
        { "field": "Email", "message": "must be a valid email address" }
    ]
}
```

Supported tags: `required`, `email`, `url`/`uri`, `uuid`, `datetime`, `min`, `max`, `gte`, `lte`, `gt`, `lt`, `len`, `oneof`. All are mapped to the corresponding OpenAPI schema properties (`format`, `minimum`, `maxLength`, `enum`, etc.).

To use a custom validator instance:

```go
api := shiftapi.New(shiftapi.WithValidator(myValidator))
```

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

### TypeScript Type Safety

ShiftAPI can generate fully-typed TypeScript clients from your Go types using the `@shiftapi/vite-plugin`. One line change on the Go side, full autocomplete on the frontend.

**Go** — use `shiftapi.ListenAndServe` instead of `http.ListenAndServe`:

```go
log.Fatal(shiftapi.ListenAndServe(":8080", api))
```

**Install the Vite plugin:**

```sh
npm install @shiftapi/vite-plugin
```

**`vite.config.ts`:**

```typescript
import shiftapi from "@shiftapi/vite-plugin";
import { defineConfig } from "vite";

export default defineConfig({
    plugins: [
        shiftapi({
            server: "./cmd/server",  // Go entry point
        }),
    ],
});
```

**Frontend** — import the typed client:

```typescript
import { client } from "@shiftapi/client";

const { data } = await client.GET("/health");
// data: { ok?: boolean }

const { data: greeting } = await client.POST("/greet", {
    body: { name: "frank" },
});
// body and response are fully typed from your Go structs
```

The plugin extracts your OpenAPI spec at build time (no running server required), generates TypeScript types via [openapi-typescript](https://github.com/openapi-ts/openapi-typescript), and serves a pre-configured [openapi-fetch](https://github.com/openapi-ts/openapi-typescript/tree/main/packages/openapi-fetch) client as a virtual module.

In dev mode, the plugin also:
- Starts the Go server automatically (`go run`)
- Auto-configures Vite's proxy to forward API requests
- Watches `.go` files — on change, restarts the server, regenerates types, and reloads the browser

**Plugin options:**

| Option | Default | Description |
|---|---|---|
| `server` | *(required)* | Go entry point (e.g. `"./cmd/server"`) |
| `baseUrl` | `"/"` | Fallback base URL for the API client |
| `goRoot` | `process.cwd()` | Go module root directory |
| `url` | `"http://localhost:8080"` | Go server address for dev proxy |

**Configuring the API base URL for production:**

In dev mode, the plugin proxies all API requests through Vite's dev server so the default `baseUrl` of `"/"` works automatically. In production, where the API runs on a different host, set the `VITE_SHIFTAPI_BASE_URL` environment variable:

```bash
# .env.production
VITE_SHIFTAPI_BASE_URL=https://api.example.com
```

This follows Vite's standard [env file](https://vite.dev/guide/env-and-mode) convention — `.env.production` is loaded during `vite build`, `.env.development` during `vite dev`, etc. The `baseUrl` plugin option serves as the fallback when the env var is not set.

The plugin automatically updates your `tsconfig.json` with the required path mapping for IDE autocomplete on first run.

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

### Development

This is a pnpm + [Turborepo](https://turbo.build) monorepo. Turbo handles the build dependency graph — running `pnpm dev` will automatically build the Vite plugin before starting the example app.

```bash
pnpm install    # install dependencies
pnpm build      # build all packages
pnpm dev        # build plugin, then start example Vite + Go app
pnpm test       # run all tests
```

[release-img]: https://img.shields.io/github/v/release/fcjr/shiftapi
[release]: https://github.com/fcjr/shiftapi/releases
[golangci-lint-img]: https://github.com/fcjr/shiftapi/workflows/go-lint/badge.svg
[golangci-lint]: https://github.com/fcjr/shiftapi/actions?query=workflow%3Ago-lint
[report-card-img]: https://goreportcard.com/badge/github.com/fcjr/shiftapi
[report-card]: https://goreportcard.com/report/github.com/fcjr/shiftapi
