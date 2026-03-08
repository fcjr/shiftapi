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
// # Error handling
//
// Use [WithError] to declare that a specific error type may be returned at a
// given HTTP status code. The error type must implement the [error] interface and
// its struct fields are reflected into the OpenAPI schema.
//
// Use [WithGlobalError] at the API level (applies to all routes) or [WithError]
// at the route level (applies to a single route):
//
//	// API-level — applies to all routes
//	api := shiftapi.New(
//	    shiftapi.WithGlobalError[*AuthError](http.StatusUnauthorized),
//	)
//
//	// Route-level — applies to this route only
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
// Use [WithBadRequestError] to customize the 400 response for parse errors:
//
//	api := shiftapi.New(
//	    shiftapi.WithBadRequestError(func(err error) *MyBadRequest {
//	        return &MyBadRequest{Code: "BAD_REQUEST", Message: err.Error()}
//	    }),
//	)
//
// Use [WithInternalServerError] to customize the 500 response body and schema:
//
//	api := shiftapi.New(
//	    shiftapi.WithInternalServerError(func(err error) *MyServerError {
//	        return &MyServerError{Code: "INTERNAL_ERROR", Message: "internal server error"}
//	    }),
//	)
//
// Every route automatically includes 422 ([ValidationError]) and 500 responses
// in the generated OpenAPI spec.
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
