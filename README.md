

<p align="center">
	<picture>
		<source media="(prefers-color-scheme: dark)" srcset="assets/logo-dark.svg">
		<img src="assets/logo.svg" alt="ShiftAPI Logo">
	</picture>
</p>

<h3 align="center">End-to-end type safety from Go structs to TypeScript frontend.</h3>

<p align="center">
  ShiftAPI is a Go framework that generates an OpenAPI 3.1 spec from your handler types at runtime, then uses a Vite or Next.js plugin to turn that spec into a fully-typed TypeScript client — so your frontend stays in sync with your API automatically.
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/fcjr/shiftapi"><img src="https://pkg.go.dev/badge/github.com/fcjr/shiftapi.svg" alt="Go Reference"></a>
  <a href="https://github.com/fcjr/shiftapi/actions?query=workflow%3Ago-lint"><img src="https://github.com/fcjr/shiftapi/workflows/ci/badge.svg" alt="GolangCI"></a>
  <a href="https://goreportcard.com/report/github.com/fcjr/shiftapi"><img src="https://goreportcard.com/badge/github.com/fcjr/shiftapi" alt="Go Report Card"></a>
  <a href="https://www.npmjs.com/package/shiftapi"><img src="https://img.shields.io/npm/v/shiftapi?label=shiftapi" alt="npm shiftapi"></a>
  <a href="https://www.npmjs.com/package/@shiftapi/vite-plugin"><img src="https://img.shields.io/npm/v/@shiftapi/vite-plugin?label=%40shiftapi%2Fvite-plugin" alt="npm @shiftapi/vite-plugin"></a>
  <a href="https://www.npmjs.com/package/@shiftapi/next"><img src="https://img.shields.io/npm/v/@shiftapi/next?label=%40shiftapi%2Fnext" alt="npm @shiftapi/next"></a>
</p>

```
Go structs ──→ OpenAPI 3.1 spec ──→ TypeScript types ──→ Typed fetch client
   (compile time)     (runtime)         (build time)        (your frontend)
```

## Getting Started

Scaffold a full-stack app (Go + React, Svelte, or Next.js):

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

    shiftapi.Handle(api, "POST /greet", greet)

    log.Println("listening on :8080")
    log.Fatal(shiftapi.ListenAndServe(":8080", api))
    // interactive docs at http://localhost:8080/docs
}
```

That's it. ShiftAPI reflects your Go types into an OpenAPI 3.1 spec at `/openapi.json` and serves interactive docs at `/docs` — no code generation step, no annotations.

## Features

### Generic type-safe handlers

Generic free functions capture your request and response types at compile time. Every method uses a single function — struct tags discriminate query params (`query:"..."`), HTTP headers (`header:"..."`), body fields (`json:"..."`), and form fields (`form:"..."`). For routes without input, use `_ struct{}`.

```go
// POST with body — input is decoded and passed as *CreateUser
shiftapi.Handle(api, "POST /users", func(r *http.Request, in *CreateUser) (*User, error) {
    return db.CreateUser(r.Context(), in)
}, shiftapi.WithStatus(http.StatusCreated))

// GET without input — use _ struct{}
shiftapi.Handle(api, "GET /users/{id}", func(r *http.Request, _ struct{}) (*User, error) {
    return db.GetUser(r.Context(), r.PathValue("id"))
})
```

### Typed path parameters

Use `path` tags to declare typed path parameters. They are parsed from the URL, validated, and documented in the OpenAPI spec automatically:

```go
type GetUserInput struct {
    ID int `path:"id" validate:"required,gt=0"`
}

shiftapi.Handle(api, "GET /users/{id}", func(r *http.Request, in GetUserInput) (*User, error) {
    return db.GetUser(r.Context(), in.ID) // in.ID is already an int
})
```

Supports the same scalar types as query params: `string`, `bool`, `int*`, `uint*`, `float*`. Use `validate:"uuid"` on a `string` field for UUID path params. Parse errors return `400`; validation failures return `422`.

Path parameters are always required and always scalar — pointers and slices on `path`-tagged fields panic at registration time. You can still use `r.PathValue("id")` directly for routes that don't need typed path params.

### Typed query parameters

Define a struct with `query` tags. Query params are parsed, validated, and documented in the OpenAPI spec automatically.

```go
type SearchQuery struct {
    Q     string `query:"q"     validate:"required"`
    Page  int    `query:"page"  validate:"min=1"`
    Limit int    `query:"limit" validate:"min=1,max=100"`
}

shiftapi.Handle(api, "GET /search", func(r *http.Request, in SearchQuery) (*Results, error) {
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

shiftapi.Handle(api, "POST /items", func(r *http.Request, in CreateInput) (*Result, error) {
    return createItem(in.Name, in.DryRun), nil
})
```

### Typed HTTP headers

Define a struct with `header` tags. Headers are parsed, validated, and documented in the OpenAPI spec automatically — just like query params.

```go
type AuthInput struct {
    Token string `header:"Authorization" validate:"required"`
    Q     string `query:"q"`
}

shiftapi.Handle(api, "GET /search", func(r *http.Request, in AuthInput) (*Results, error) {
    // in.Token parsed from the Authorization header
    // in.Q parsed from ?q= query param
    return doSearch(in.Token, in.Q), nil
})
```

Supports `string`, `bool`, `int*`, `uint*`, `float*` scalars and `*T` pointers for optional headers. Parse errors return `400`; validation failures return `422`. Header, query, and body fields can be freely combined in one struct.

### File uploads (`multipart/form-data`)

Use `form` tags to declare file upload endpoints. The `form` tag drives OpenAPI spec generation — the generated TypeScript client gets the correct `multipart/form-data` types automatically. At runtime, the request body is parsed via `ParseMultipartForm` and form-tagged fields are populated.

```go
type UploadInput struct {
    File  *multipart.FileHeader   `form:"file" validate:"required"`
    Title string                  `form:"title" validate:"required"`
    Tags  string                  `query:"tags"`
}

shiftapi.Handle(api, "POST /upload", func(r *http.Request, in UploadInput) (*Result, error) {
    f, err := in.File.Open()
    if err != nil {
        return nil, fmt.Errorf("failed to open file: %w", err)
    }
    defer f.Close()
    // read from f, save to disk/S3/etc.
    return &Result{Filename: in.File.Filename, Title: in.Title}, nil
})
```

- `*multipart.FileHeader` — single file (`type: string, format: binary` in OpenAPI, `File | Blob | Uint8Array` in TypeScript)
- `[]*multipart.FileHeader` — multiple files (`type: array, items: {type: string, format: binary}`)
- Scalar types with `form` tag — text form fields
- `query` tags work alongside `form` tags
- Mixing `json` and `form` tags on the same struct panics at registration time

Restrict accepted file types with the `accept` tag. This validates the `Content-Type` at runtime (returns `400` if rejected) and documents the constraint in the OpenAPI spec via the `encoding` map:

```go
type ImageUpload struct {
    Avatar *multipart.FileHeader `form:"avatar" accept:"image/png,image/jpeg" validate:"required"`
}
```

The default max upload size is 32 MB. Configure it with `WithMaxUploadSize`:

```go
api := shiftapi.New(shiftapi.WithMaxUploadSize(64 << 20)) // 64 MB
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

### Route groups

Use `Group` to create a sub-router with a shared path prefix and options. Groups can be nested:

```go
v1 := api.Group("/api/v1",
    shiftapi.WithError[*RateLimitError](http.StatusTooManyRequests),
    shiftapi.WithMiddleware(auth),
)

shiftapi.Handle(v1, "GET /users", listUsers)   // GET /api/v1/users
shiftapi.Handle(v1, "POST /users", createUser) // POST /api/v1/users

admin := v1.Group("/admin",
    shiftapi.WithError[*ForbiddenError](http.StatusForbidden),
    shiftapi.WithMiddleware(adminOnly),
)
shiftapi.Handle(admin, "GET /stats", getStats) // GET /api/v1/admin/stats
```

### Middleware

Use `WithMiddleware` to apply standard HTTP middleware at any level — API, group, or route:

```go
api := shiftapi.New(
    shiftapi.WithMiddleware(cors, logging),          // all routes
)
v1 := api.Group("/api/v1",
    shiftapi.WithMiddleware(auth),                   // group routes
)
shiftapi.Handle(v1, "GET /admin", getAdmin,
    shiftapi.WithMiddleware(adminOnly),               // single route
)
```

Middleware resolves from outermost to innermost: **API → parent Group → child Group → Route → handler**. Within a single `WithMiddleware(a, b)` call, the first argument wraps outermost.

### Context values

Use `NewContextKey`, `SetContext`, and `FromContext` to pass typed data from middleware to handlers — no untyped `context.Value` keys or type assertions needed:

```go
var userKey = shiftapi.NewContextKey[User]("user")

// Middleware stores the value:
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user, err := authenticate(r)
        if err != nil {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, shiftapi.SetContext(r, userKey, user))
    })
}

// Handler retrieves it — fully typed, no assertion needed:
shiftapi.Handle(authed, "GET /me", func(r *http.Request, _ struct{}) (*Profile, error) {
    user, ok := shiftapi.FromContext(r, userKey)
    if !ok {
        return nil, fmt.Errorf("missing user context")
    }
    return &Profile{Name: user.Name}, nil
})
```

Each `ContextKey` has pointer identity, so two keys for the same type never collide. The type parameter ensures `SetContext` and `FromContext` agree on the value type at compile time.

### Error handling

Use `WithError` to declare that a handler may return a specific error type at a given HTTP status code. Works at any level — API, group, or route:

```go
api := shiftapi.New(
    shiftapi.WithError[*AuthError](http.StatusUnauthorized),         // all routes
)
shiftapi.Handle(api, "GET /users/{id}", getUser,
    shiftapi.WithError[*NotFoundError](http.StatusNotFound),         // single route
)
```

The error type must implement `error` — its struct fields are reflected into the OpenAPI schema. At runtime, if the handler returns a matching error (via `errors.As`), it is serialized as JSON with the declared status code. Wrapped errors work automatically. Unrecognized errors return `500`.

Customize the default 400/500 responses with `WithBadRequestError` and `WithInternalServerError`:

```go
api := shiftapi.New(
    shiftapi.WithBadRequestError(func(err error) *MyBadRequest {
        return &MyBadRequest{Code: "BAD_REQUEST", Message: err.Error()}
    }),
    shiftapi.WithInternalServerError(func(err error) *MyServerError {
        log.Error("unhandled", "err", err)
        return &MyServerError{Code: "INTERNAL_ERROR", Message: "internal server error"}
    }),
)
```

Every route automatically includes `400`, `422` ([ValidationError](https://pkg.go.dev/github.com/fcjr/shiftapi#ValidationError)), and `500` responses in the generated OpenAPI spec.

### Option composition

`WithError` and `WithMiddleware` are `Option` values — they work at all three levels. Use `ComposeOptions` to bundle them into reusable options:

```go
func WithAuth() shiftapi.Option {
    return shiftapi.ComposeOptions(
        shiftapi.WithMiddleware(authMiddleware),
        shiftapi.WithError[*AuthError](http.StatusUnauthorized),
    )
}
```

For level-specific composition (mixing shared and level-specific options), use `ComposeAPIOptions`, `ComposeGroupOptions`, or `ComposeRouteOptions`:

```go
createOpts := shiftapi.ComposeRouteOptions(
    shiftapi.WithStatus(http.StatusCreated),
    shiftapi.WithError[*ConflictError](http.StatusConflict),
)
shiftapi.Handle(api, "POST /users", createUser, createOpts)
```

### Route metadata

Add OpenAPI summaries, descriptions, and tags per route:

```go
shiftapi.Handle(api, "POST /greet", greet,
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

ShiftAPI ships npm packages for the frontend:

- **`shiftapi`** — CLI and codegen core. Extracts the OpenAPI spec from your Go server, generates TypeScript types via [openapi-typescript](https://github.com/openapi-ts/openapi-typescript), and writes a pre-configured [openapi-fetch](https://github.com/openapi-ts/openapi-typescript/tree/main/packages/openapi-fetch) client.
- **`@shiftapi/vite-plugin`** — Vite plugin for dev-time HMR, proxy, and Go server management.
- **`@shiftapi/next`** — Next.js integration with the same DX (webpack/Turbopack aliases, rewrites proxy, Go server management).

**`shiftapi.config.ts`** (project root):

```typescript
import { defineConfig } from "shiftapi";

export default defineConfig({
    server: "./cmd/server", // Go entry point
});
```

### Vite

```sh
npm install shiftapi @shiftapi/vite-plugin
```

```typescript
// vite.config.ts
import shiftapi from "@shiftapi/vite-plugin";
import { defineConfig } from "vite";

export default defineConfig({
    plugins: [shiftapi()],
});
```

### Next.js

```sh
npm install shiftapi @shiftapi/next
```

```typescript
// next.config.ts
import type { NextConfig } from "next";
import { withShiftAPI } from "@shiftapi/next";

const nextConfig: NextConfig = {};

export default withShiftAPI(nextConfig);
```

### Use the typed client

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

const { data: upload } = await client.POST("/upload", {
    body: { file: new File(["content"], "doc.txt"), title: "My Doc" },
    params: { query: { tags: "important" } },
});
// file uploads are typed as File | Blob | Uint8Array — generated from format: binary in the spec

const { data: authResults } = await client.GET("/search", {
    params: {
        query: { q: "hello" },
        header: { Authorization: "Bearer token" },
    },
});
// header params are fully typed as well
```

In dev mode the plugins start the Go server, proxy API requests, watch `.go` files, and regenerate types on changes.

**CLI usage (without Vite/Next.js):**

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

For production, set `VITE_SHIFTAPI_BASE_URL` (Vite) or `NEXT_PUBLIC_SHIFTAPI_BASE_URL` (Next.js) to point at your API host. The plugins automatically update `tsconfig.json` with the required path mapping for IDE autocomplete.

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

---

<p align="center">Made with love for types at the <a href="https://www.recurse.com">Recurse Center</a></p>
