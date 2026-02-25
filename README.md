
<p align="center">
	<picture>
		<source media="(prefers-color-scheme: dark)" srcset="assets/logo-dark.svg">
		<img src="assets/logo.svg" alt="ShiftAPI Logo">
	</picture>
</p>

<h3 align="center">End-to-end type safety from Go structs to TypeScript frontend.</h3>

<p align="center">
  ShiftAPI is a Go framework that generates an OpenAPI 3.1 spec from your handler types at runtime, then uses a Vite plugin to turn that spec into a fully-typed TypeScript client — so your frontend stays in sync with your API automatically.
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/fcjr/shiftapi"><img src="https://pkg.go.dev/badge/github.com/fcjr/shiftapi.svg" alt="Go Reference"></a>
  <a href="https://github.com/fcjr/shiftapi/actions?query=workflow%3Ago-lint"><img src="https://github.com/fcjr/shiftapi/workflows/go-lint/badge.svg" alt="GolangCI"></a>
  <a href="https://goreportcard.com/report/github.com/fcjr/shiftapi"><img src="https://goreportcard.com/badge/github.com/fcjr/shiftapi" alt="Go Report Card"></a>
  <a href="https://www.npmjs.com/package/shiftapi"><img src="https://img.shields.io/npm/v/shiftapi" alt="npm shiftapi"></a>
  <a href="https://www.npmjs.com/package/@shiftapi/vite-plugin"><img src="https://img.shields.io/npm/v/@shiftapi/vite-plugin" alt="npm @shiftapi/vite-plugin"></a>
</p>

```
Go structs ──→ OpenAPI 3.1 spec ──→ TypeScript types ──→ Typed fetch client
   (compile time)     (runtime)         (build time)        (your frontend)
```

## Getting Started

Scaffold a full-stack app (Go + React or Svelte):

```sh
npm create shiftapi@latest
```

Or add ShiftAPI to an existing Go project:

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

func greet(r *http.Request, in *Person) (*Greeting, error) {
    return &Greeting{Hello: in.Name}, nil
}

func main() {
    api := shiftapi.New(shiftapi.WithInfo(shiftapi.Info{
        Title:   "Greeter API",
        Version: "1.0.0",
    }))

    shiftapi.Post(api, "/greet", greet)

    log.Println("listening on :8080")
    log.Fatal(shiftapi.ListenAndServe(":8080", api))
    // interactive docs at http://localhost:8080/docs
}
```

That's it. ShiftAPI reflects your Go types into an OpenAPI 3.1 spec at `/openapi.json` and serves interactive docs at `/docs` — no code generation step, no annotations.

## Features

### Generic type-safe handlers

Generic free functions capture your request and response types at compile time. Every method uses a single function — struct tags discriminate query params (`query:"..."`) from body fields (`json:"..."`). For routes without input, use `_ struct{}`.

```go
// POST with body — input is decoded and passed as *CreateUser
shiftapi.Post(api, "/users", func(r *http.Request, in *CreateUser) (*User, error) {
    return db.CreateUser(r.Context(), in)
}, shiftapi.WithStatus(http.StatusCreated))

// GET without input — use _ struct{}
shiftapi.Get(api, "/users/{id}", func(r *http.Request, _ struct{}) (*User, error) {
    return db.GetUser(r.Context(), r.PathValue("id"))
})
```

### Typed query parameters

Define a struct with `query` tags. Query params are parsed, validated, and documented in the OpenAPI spec automatically.

```go
type SearchQuery struct {
    Q     string `query:"q"     validate:"required"`
    Page  int    `query:"page"  validate:"min=1"`
    Limit int    `query:"limit" validate:"min=1,max=100"`
}

shiftapi.Get(api, "/search", func(r *http.Request, in SearchQuery) (*Results, error) {
    return doSearch(in.Q, in.Page, in.Limit), nil
})
```

Supports `string`, `bool`, `int*`, `uint*`, `float*` scalars, `*T` pointers for optional params, and `[]T` slices for repeated params (e.g. `?tag=a&tag=b`). Parse errors return `400`; validation failures return `422`.

For handlers that need both query parameters and a request body, combine them in a single struct — fields with `query` tags become query params, fields with `json` tags become the body:

```go
type CreateInput struct {
    DryRun bool   `query:"dry_run"`
    Name   string `json:"name"`
}

shiftapi.Post(api, "/items", func(r *http.Request, in CreateInput) (*Result, error) {
    return createItem(in.Name, in.DryRun), nil
})
```

### Validation

Built-in validation via [go-playground/validator](https://github.com/go-playground/validator). Struct tags are enforced at runtime *and* reflected into the OpenAPI schema.

```go
type CreateUser struct {
    Name  string `json:"name"  validate:"required,min=2,max=50"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age"   validate:"gte=0,lte=150"`
    Role  string `json:"role"  validate:"oneof=admin user guest"`
}
```

Invalid requests return `422` with per-field errors:

```json
{
    "message": "validation failed",
    "errors": [
        { "field": "Name",  "message": "this field is required" },
        { "field": "Email", "message": "must be a valid email address" }
    ]
}
```

Supported tags: `required`, `email`, `url`/`uri`, `uuid`, `datetime`, `min`, `max`, `gte`, `lte`, `gt`, `lt`, `len`, `oneof` — all mapped to their OpenAPI equivalents (`format`, `minimum`, `maxLength`, `enum`, etc.). Use `WithValidator()` to supply a custom validator instance.

### Error handling

Return `shiftapi.Error` to control the status code:

```go
return nil, shiftapi.Error(http.StatusNotFound, "user not found")
```

Any non-`APIError` returns `500 Internal Server Error`.

### Route metadata

Add OpenAPI summaries, descriptions, and tags per route:

```go
shiftapi.Post(api, "/greet", greet,
    shiftapi.WithRouteInfo(shiftapi.RouteInfo{
        Summary:     "Greet a person",
        Description: "Returns a personalized greeting.",
        Tags:        []string{"greetings"},
    }),
)
```

### Standard `http.Handler`

`API` implements `http.Handler`, so it works with any middleware, `httptest`, and `ServeMux` mounting:

```go
// middleware
wrapped := loggingMiddleware(corsMiddleware(api))
http.ListenAndServe(":8080", wrapped)

// mount under a prefix
mux := http.NewServeMux()
mux.Handle("/api/v1/", http.StripPrefix("/api/v1", api))
```

## TypeScript Integration

ShiftAPI ships two npm packages for the frontend:

- **`shiftapi`** — CLI and codegen core. Extracts the OpenAPI spec from your Go server, generates TypeScript types via [openapi-typescript](https://github.com/openapi-ts/openapi-typescript), and writes a pre-configured [openapi-fetch](https://github.com/openapi-ts/openapi-typescript/tree/main/packages/openapi-fetch) client.
- **`@shiftapi/vite-plugin`** — Vite plugin that wraps the CLI for dev-time HMR, proxy, and Go server management.

**Install:**

```sh
npm install shiftapi @shiftapi/vite-plugin
```

**`shiftapi.config.ts`** (project root):

```typescript
import { defineConfig } from "shiftapi";

export default defineConfig({
    server: "./cmd/server", // Go entry point
});
```

**`vite.config.ts`:**

```typescript
import shiftapi from "@shiftapi/vite-plugin";
import { defineConfig } from "vite";

export default defineConfig({
    plugins: [shiftapi()],
});
```

**Use the typed client:**

```typescript
import { client } from "@shiftapi/client";

const { data } = await client.GET("/health");
// data: { ok?: boolean }

const { data: greeting } = await client.POST("/greet", {
    body: { name: "frank" },
});
// body and response are fully typed from your Go structs

const { data: results } = await client.GET("/search", {
    params: { query: { q: "hello", page: 1, limit: 10 } },
});
// query params are fully typed too — { q: string, page?: number, limit?: number }
```

In dev mode the plugin also starts the Go server, proxies API requests through Vite, watches `.go` files, and hot-reloads the frontend when types change.

**CLI usage (without Vite):**

```sh
shiftapi prepare
```

This extracts the spec and generates `.shiftapi/client.d.ts` and `.shiftapi/client.js`. Useful in `postinstall` scripts or CI.

**Config options:**

| Option | Default | Description |
|---|---|---|
| `server` | *(required)* | Go entry point (e.g. `"./cmd/server"`) |
| `baseUrl` | `"/"` | Fallback base URL for the API client |
| `url` | `"http://localhost:8080"` | Go server address for dev proxy |

For production, set `VITE_SHIFTAPI_BASE_URL` in a `.env.production` file to point at your API host. The plugin automatically updates `tsconfig.json` with the required path mapping for IDE autocomplete.

## Development

This is a pnpm + [Turborepo](https://turbo.build) monorepo.

```bash
pnpm install    # install dependencies
pnpm build      # build all packages
pnpm dev        # start example Vite + Go app
pnpm test       # run all tests
```

Go tests can also be run directly:

```bash
go test -count=1 -tags shiftapidev ./...
```
