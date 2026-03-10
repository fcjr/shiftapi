// Package shiftapi provides end-to-end type safety from Go to TypeScript.
//
// Define your API with typed Go handler functions and shiftapi automatically
// generates an OpenAPI 3.1 spec, validates requests, and produces a
// fully-typed TypeScript client — all from a single source of truth.
//
// # Quick start
//
//	api := shiftapi.New()
//	shiftapi.Handle(api, "POST /greet", greet)
//	shiftapi.ListenAndServe(":8080", api)
//
// where greet is a typed handler:
//
//	type GreetRequest struct {
//	    Name string `json:"name" validate:"required"`
//	}
//
//	type GreetResponse struct {
//	    Hello string `json:"hello"`
//	}
//
//	func greet(r *http.Request, in *GreetRequest) (*GreetResponse, error) {
//	    return &GreetResponse{Hello: in.Name}, nil
//	}
//
// # Struct tag conventions
//
// ShiftAPI discriminates input struct fields by their struct tags:
//
//   - path:"name" — parsed from URL path parameters (e.g. /users/{id})
//   - json:"name" — parsed from the JSON request body (default for POST/PUT/PATCH)
//   - query:"name" — parsed from URL query parameters
//   - header:"name" — parsed from HTTP request headers (input) or set as HTTP
//     response headers (output)
//   - form:"name" — parsed from multipart/form-data (for file uploads)
//   - validate:"rules" — validated using [github.com/go-playground/validator/v10]
//     rules and reflected into the OpenAPI schema
//   - accept:"mime/type" — constrains accepted MIME types on form file fields
//
// A single input struct can mix path, query, and body fields:
//
//	type GetUserRequest struct {
//	    ID     int    `path:"id" validate:"required,gt=0"`
//	    Fields string `query:"fields"`
//	}
//
// # Enums
//
// Use [WithEnum] to register the allowed values for a named type. The values
// are reflected as an enum constraint in the OpenAPI schema for every field of
// that type — no validate:"oneof=..." tag required:
//
//	type Status string
//
//	const (
//	    StatusActive   Status = "active"
//	    StatusInactive Status = "inactive"
//	    StatusPending  Status = "pending"
//	)
//
//	api := shiftapi.New(
//	    shiftapi.WithEnum[Status](StatusActive, StatusInactive, StatusPending),
//	)
//
// Enum values apply to body fields, query parameters, path parameters, and
// header parameters. If a field also carries a validate:"oneof=..." tag, the
// tag takes precedence over the registered enum values.
//
// The type parameter must satisfy the [Scalar] constraint (~string, ~int*,
// ~uint*, ~float*).
//
// # File uploads
//
// Use [*multipart.FileHeader] fields with the form tag for file uploads:
//
//	type UploadInput struct {
//	    File *multipart.FileHeader   `form:"file" validate:"required"`
//	    Docs []*multipart.FileHeader `form:"docs"`
//	}
//
// # Response headers
//
// Use the header tag on the Resp struct to set HTTP response headers.
// Header-tagged fields are written as response headers and automatically
// excluded from the JSON response body. They are also documented as response
// headers in the OpenAPI spec.
//
//	type CachedResponse struct {
//	    CacheControl string  `header:"Cache-Control"`
//	    ETag         *string `header:"ETag"`            // optional — omitted when nil
//	    Items        []Item  `json:"items"`
//	}
//
// Non-pointer fields are always sent, even with a zero value. Use a pointer
// field for optional headers that should only be sent when set. Supported
// types are the same scalars as request headers (string, bool, int*, uint*,
// float*).
//
// For static response headers, use [WithResponseHeader]. Headers are applied
// in the following order — later sources override earlier ones for the same
// header name:
//
//  1. Middleware-set headers (outermost, applied before the handler)
//  2. Static headers via [WithResponseHeader] (API → Group → Route)
//  3. Dynamic headers via header struct tags (innermost, applied last)
//
// # No-body responses
//
// For status codes that forbid a response body (204 No Content, 304 Not Modified),
// use [WithStatus] with struct{} or a header-only response type. No JSON body or
// Content-Type header will be written. Response headers (both static and dynamic)
// are still sent.
//
//	shiftapi.Handle(api, "DELETE /items/{id}", deleteItem,
//	    shiftapi.WithStatus(http.StatusNoContent),
//	)
//
// Registering a route with status 204 or 304 and a response type that has JSON body
// fields panics at startup — this catches misconfigurations early.
//
// # Server-Sent Events
//
// Use [HandleSSE] for Server-Sent Events with a typed event writer:
//
//	type ChatEvent struct {
//	    User    string `json:"user"`
//	    Message string `json:"message"`
//	}
//
//	shiftapi.HandleSSE(api, "GET /chat", func(r *http.Request, _ struct{}, sse *shiftapi.SSEWriter[ChatEvent]) error {
//	    for event := range events(r.Context()) {
//	        if err := sse.Send(event); err != nil {
//	            return err
//	        }
//	    }
//	    return nil
//	})
//
// [SSEWriter] automatically sets Content-Type, Cache-Control, and Connection
// headers on the first write. [SSEWriter.Send] automatically determines the
// event name from the concrete Go type registered via [SSESends].
//
// [SSESends] is required for [HandleSSE]. It registers event types for
// auto-wrap and OpenAPI schema generation. For multiple event types, define
// a marker interface:
//
//	type ChatEvent interface{ chatEvent() }
//
//	type MessageData struct {
//	    User string `json:"user"`
//	    Text string `json:"text"`
//	}
//	func (MessageData) chatEvent() {}
//
//	type JoinData struct {
//	    User string `json:"user"`
//	}
//	func (JoinData) chatEvent() {}
//
//	shiftapi.HandleSSE(api, "GET /chat", func(r *http.Request, _ struct{}, sse *shiftapi.SSEWriter[ChatEvent]) error {
//	    sse.Send(MessageData{User: "alice", Text: "hi"})
//	    return sse.Send(JoinData{User: "bob"})
//	}, shiftapi.SSESends(
//	    shiftapi.SSEEventType[MessageData]("message"),
//	    shiftapi.SSEEventType[JoinData]("join"),
//	))
//
// [SSESends] generates a oneOf schema with a discriminator in the OpenAPI spec,
// which produces TypeScript discriminated unions in the generated client.
//
// The generated TypeScript client includes a typed subscribe function
// constrained to SSE paths. It handles path/query/header parameter
// substitution, SSE stream parsing, and yields typed events as an async
// iterable.
//
// For custom SSE framing or non-standard behavior, use [HandleRaw] with
// [WithContentType]("text/event-stream") instead.
//
// # Route groups
//
// Use [API.Group] to create a sub-router with a shared path prefix and options.
// Groups can be nested, and error types and middleware are inherited by child groups:
//
//	v1 := api.Group("/api/v1",
//	    shiftapi.WithMiddleware(auth),
//	)
//	shiftapi.Handle(v1, "GET /users", listUsers) // registers GET /api/v1/users
//
//	admin := v1.Group("/admin",
//	    shiftapi.WithError[*ForbiddenError](http.StatusForbidden),
//	)
//	shiftapi.Handle(admin, "GET /stats", getStats) // registers GET /api/v1/admin/stats
//
// # Middleware
//
// Use [WithMiddleware] to apply standard HTTP middleware at any level:
//
//	api := shiftapi.New(
//	    shiftapi.WithMiddleware(cors, logging),          // all routes
//	)
//	v1 := api.Group("/api/v1",
//	    shiftapi.WithMiddleware(auth),                   // group routes
//	)
//	shiftapi.Handle(v1, "GET /admin", getAdmin,
//	    shiftapi.WithMiddleware(adminOnly),               // single route
//	)
//
// Middleware is applied from outermost to innermost in the order:
// API → parent Group → child Group → Route → handler.
// Within a single [WithMiddleware] call, the first argument wraps outermost.
//
// # Context values
//
// Use [NewContextKey], [SetContext], and [FromContext] to pass typed data from
// middleware to handlers without untyped context keys or type assertions:
//
//	var userKey = shiftapi.NewContextKey[User]("user")
//
//	// Middleware stores the value:
//	func authMiddleware(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        user := authenticate(r)
//	        next.ServeHTTP(w, shiftapi.SetContext(r, userKey, user))
//	    })
//	}
//
//	// Handler retrieves it — fully typed, no assertion needed:
//	func getProfile(r *http.Request, _ struct{}) (*Profile, error) {
//	    user, ok := shiftapi.FromContext(r, userKey)
//	    if !ok {
//	        return nil, errUnauthorized
//	    }
//	    return &Profile{Name: user.Name}, nil
//	}
//
// Each [ContextKey] has pointer identity, so two keys for the same type T will
// never collide. The type parameter ensures that [SetContext] and [FromContext]
// agree on the value type at compile time.
//
// # Error handling
//
// Use [WithError] to declare that a specific error type may be returned at a
// given HTTP status code. The error type must implement the [error] interface and
// its struct fields are reflected into the OpenAPI schema. [WithError] works at
// all three levels: [New], [API.Group]/[Group.Group], and route functions.
//
//	api := shiftapi.New(
//	    shiftapi.WithError[*AuthError](http.StatusUnauthorized),
//	)
//	shiftapi.Handle(api, "GET /users/{id}", getUser,
//	    shiftapi.WithError[*NotFoundError](http.StatusNotFound),
//	)
//
// At runtime, if the handler returns an error matching a registered type (via
// [errors.As]), it is serialized as JSON with the declared status code. Multiple
// error types can be declared per route. Wrapped errors are matched automatically.
//
// Validation failures automatically return 422 with structured [ValidationError] responses.
// Unrecognized errors return 500 Internal Server Error to prevent leaking implementation details.
//
// Use [WithBadRequestError] and [WithInternalServerError] to customize the default
// 400 and 500 response bodies.
//
// # Options
//
// [Option] is the primary option type. It works at all three levels: [New],
// [API.Group]/[Group.Group], and [Handle].
// [WithError], [WithMiddleware], and [WithResponseHeader] all return [Option].
//
// Some options are level-specific: [WithInfo] and [WithBadRequestError] only work
// with [New] ([APIOption]), while [WithStatus] and [WithRouteInfo] only work with
// [Handle] ([RouteOption]).
//
// Use [ComposeOptions] to bundle multiple [Option] values into a reusable option:
//
//	func WithAuth() shiftapi.Option {
//	    return shiftapi.ComposeOptions(
//	        shiftapi.WithMiddleware(authMiddleware),
//	        shiftapi.WithError[*AuthError](http.StatusUnauthorized),
//	    )
//	}
//
// [ComposeAPIOptions], [ComposeGroupOptions], and [ComposeRouteOptions] can mix shared and
// level-specific options at their respective levels.
//
// # Built-in endpoints
//
// Every API automatically serves:
//
//   - GET /openapi.json — the generated OpenAPI 3.1 spec
//   - GET /docs — interactive API documentation (Scalar UI)
//
// # http.Handler compatibility
//
// [API] implements [http.Handler], so it works with any standard middleware,
// router, or test framework:
//
//	mux := http.NewServeMux()
//	mux.Handle("/api/", http.StripPrefix("/api", api))
package shiftapi
