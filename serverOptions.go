package shiftapi

import (
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

// Info describes the API and is rendered into the OpenAPI spec's info object.
type Info struct {
	Title          string
	Description    string
	TermsOfService string
	Contact        *Contact
	License        *License
	Version        string
}

// Contact describes the API contact information.
type Contact struct {
	Name  string
	URL   string
	Email string
}

// License describes the API license.
type License struct {
	Name string
	URL  string
}

// ExternalDocs links to external documentation.
type ExternalDocs struct {
	Description string
	URL         string
}

// WithInfo configures the API metadata that appears in the OpenAPI spec
// and documentation UI.
//
//	api := shiftapi.New(shiftapi.WithInfo(shiftapi.Info{
//	    Title:   "My API",
//	    Version: "1.0.0",
//	}))
func WithInfo(info Info) apiOptionFunc {
	return func(api *API) {
		api.spec.Info = &openapi3.Info{
			Title:          info.Title,
			Description:    info.Description,
			TermsOfService: info.TermsOfService,
			Version:        info.Version,
		}
		if info.Contact != nil {
			api.spec.Info.Contact = &openapi3.Contact{
				Name:  info.Contact.Name,
				URL:   info.Contact.URL,
				Email: info.Contact.Email,
			}
		}
		if info.License != nil {
			api.spec.Info.License = &openapi3.License{
				Name: info.License.Name,
				URL:  info.License.URL,
			}
		}
	}
}

// WithMaxUploadSize sets the maximum memory used for parsing multipart form data.
// The default is 32 MB.
func WithMaxUploadSize(size int64) apiOptionFunc {
	return func(api *API) {
		api.maxUploadSize = size
	}
}

// WithBadRequestError customizes the 400 Bad Request response returned when
// the framework cannot parse the request (malformed JSON, invalid query
// parameters, invalid form data). The function receives the parse error and
// returns the value to serialize as the response body. T's type determines the
// BadRequestError schema in the OpenAPI spec.
//
//	api := shiftapi.New(
//	    shiftapi.WithBadRequestError(func(err error) *MyBadRequest {
//	        return &MyBadRequest{Code: "BAD_REQUEST", Message: err.Error()}
//	    }),
//	)
func WithBadRequestError[T any](fn func(error) T) apiOptionFunc {
	return func(api *API) {
		api.badRequestFn = func(err error) any { return fn(err) }
		registerErrorSchema[T](api, "BadRequestError")
	}
}

// WithInternalServerError customizes the 500 Internal Server Error response
// returned when a handler returns an error that doesn't match any registered
// error type. The function receives the unhandled error and returns the value
// to serialize as the response body. T's type determines the InternalServerError
// schema in the OpenAPI spec.
//
//	api := shiftapi.New(
//	    shiftapi.WithInternalServerError(func(err error) *MyServerError {
//	        log.Error("unhandled", "err", err)
//	        return &MyServerError{Code: "INTERNAL_ERROR", Message: "internal server error"}
//	    }),
//	)
func WithInternalServerError[T any](fn func(error) T) apiOptionFunc {
	return func(api *API) {
		api.internalServerFn = func(err error) any { return fn(err) }
		registerErrorSchema[T](api, "InternalServerError")
	}
}

// registerErrorSchema generates and registers a component schema for the given type.
func registerErrorSchema[T any](api *API, name string) {
	t := reflect.TypeFor[T]()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	gen := openapi3gen.NewGenerator(
		openapi3gen.SchemaCustomizer(api.schemaCustomizer),
	)
	schema, err := gen.GenerateSchemaRef(t)
	if err != nil {
		panic("shiftapi: failed to generate " + name + " schema: " + err.Error())
	}
	api.spec.Components.Schemas[name] = &openapi3.SchemaRef{
		Value: schema.Value,
	}
}

// WithExternalDocs links to external documentation.
func WithExternalDocs(docs ExternalDocs) apiOptionFunc {
	return func(api *API) {
		api.spec.ExternalDocs = &openapi3.ExternalDocs{
			Description: docs.Description,
			URL:         docs.URL,
		}
	}
}
