// Package shiftapi provides end-to-end type safety from Go to TypeScript.
//
// Define your API with typed Go handler functions and shiftapi automatically
// generates an OpenAPI 3.1 spec, validates requests, and produces a
// fully-typed TypeScript client — all from a single source of truth.
//
// # Quick start
//
//	api := shiftapi.New()
//	shiftapi.Post(api, "/greet", greet)
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
//   - json:"name" — parsed from the JSON request body (default for POST/PUT/PATCH)
//   - query:"name" — parsed from URL query parameters
//   - form:"name" — parsed from multipart/form-data (for file uploads)
//   - validate:"rules" — validated using [github.com/go-playground/validator/v10]
//     rules and reflected into the OpenAPI schema
//   - accept:"mime/type" — constrains accepted MIME types on form file fields
//
// A single input struct can mix query and body fields:
//
//	type SearchRequest struct {
//	    Q     string `query:"q" validate:"required"`
//	    Page  int    `query:"page"`
//	    Body  Filter `json:"filter"`
//	}
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
// # Route groups
//
// Use [API.Group] to create a sub-router with a shared path prefix and options.
// Groups can be nested, and error types and middleware are inherited by child groups:
//
//	v1 := api.Group("/api/v1",
//	    shiftapi.WithMiddleware(auth),
//	)
//	shiftapi.Get(v1, "/users", listUsers) // registers GET /api/v1/users
//
//	admin := v1.Group("/admin",
//	    shiftapi.WithError[*ForbiddenError](http.StatusForbidden),
//	)
//	shiftapi.Get(admin, "/stats", getStats) // registers GET /api/v1/admin/stats
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
//	shiftapi.Get(v1, "/admin", getAdmin,
//	    shiftapi.WithMiddleware(adminOnly),               // single route
//	)
//
// Middleware is applied from outermost to innermost in the order:
// API → parent Group → child Group → Route → handler.
// Within a single [WithMiddleware] call, the first argument wraps outermost.
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
//	shiftapi.Get(api, "/users/{id}", getUser,
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
// [API.Group]/[Group.Group], and route registration functions ([Get], [Post], etc.).
// [WithError] and [WithMiddleware] both return [Option].
//
// Some options are level-specific: [WithInfo] and [WithBadRequestError] only work
// with [New] ([APIOption]), while [WithStatus] and [WithRouteInfo] only work with
// route registration functions ([RouteOption]).
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
