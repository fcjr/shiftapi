package shiftapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"reflect"
)

// HandlerFunc is a typed handler function for API routes. The type parameters
// In and Resp are the request and response types — both are automatically
// reflected into the OpenAPI schema.
//
// The In struct's fields are discriminated by struct tags:
//   - path:"name" — parsed from URL path parameters (e.g. /users/{id})
//   - query:"name" — parsed from URL query parameters
//   - json:"name" — parsed from the JSON request body (default for POST/PUT/PATCH)
//   - form:"name" — parsed from multipart/form-data (for file uploads)
//
// Use struct{} as In for routes that take no input, or as Resp for routes
// that return no body (e.g. health checks that only need a status code).
//
// The [*http.Request] parameter gives access to headers, cookies, path
// parameters, and other request metadata.
type HandlerFunc[In, Resp any] func(r *http.Request, in In) (Resp, error)

func adapt[In, Resp any](fn HandlerFunc[In, Resp], status int, validate func(any) error, hasPath, hasQuery, hasBody, hasForm bool, maxUploadSize int64, errLookup errorLookup, badRequestFn, internalServerFn func(error) any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in In
		rv := reflect.ValueOf(&in).Elem()

		if hasForm {
			// Parse multipart form
			if err := parseFormInto(rv, r, maxUploadSize); err != nil {
				writeJSON(w, http.StatusBadRequest, badRequestFn(err))
				return
			}
			rv = reflect.ValueOf(&in).Elem()
		} else if hasBody {
			// JSON-decode body if there are body fields
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				writeJSON(w, http.StatusBadRequest, badRequestFn(err))
				return
			}
			// Re-point rv after decode (in case In is a pointer that was nil)
			rv = reflect.ValueOf(&in).Elem()

			// Reset any query/path-tagged fields that body decode may have
			// inadvertently set, so they only come from their proper source.
			if hasQuery {
				resetQueryFields(rv)
			}
			if hasPath {
				resetPathFields(rv)
			}
		}

		// Parse query params if there are query fields
		if hasQuery {
			if err := parseQueryInto(rv, r.URL.Query()); err != nil {
				writeJSON(w, http.StatusBadRequest, badRequestFn(err))
				return
			}
		}

		// Parse path params if there are path fields
		if hasPath {
			if err := parsePathInto(rv, r); err != nil {
				writeJSON(w, http.StatusBadRequest, badRequestFn(err))
				return
			}
		}

		if err := validate(in); err != nil {
			handleError(w, internalServerFn, err, errLookup)
			return
		}

		resp, err := fn(r, in)
		if err != nil {
			handleError(w, internalServerFn, err, errLookup)
			return
		}
		writeJSON(w, status, resp)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("shiftapi: error encoding response: %v", err)
	}
}

// handleError matches the returned error against registered error types and
// writes the appropriate HTTP response. It checks ValidationError first (always
// 422), then walks the error chain once checking each error's concrete type
// against the lookup map, and falls back to a 500 response built by
// internalServerFn.
func handleError(w http.ResponseWriter, internalServerFn func(error) any, err error, lookup errorLookup) {
	// Always check ValidationError first.
	if valErr, ok := errors.AsType[*ValidationError](err); ok {
		writeJSON(w, http.StatusUnprocessableEntity, valErr)
		return
	}

	// Walk the error chain once, checking each error's type against the map.
	if len(lookup) > 0 {
		if status, matched, ok := matchError(err, lookup); ok {
			writeJSON(w, status, matched)
			return
		}
	}

	// Fallback — 500 with the configured internal server error response.
	writeJSON(w, http.StatusInternalServerError, internalServerFn(err))
}

// matchError walks the error chain (including multi-errors) and returns the
// first error whose concrete type matches the lookup map.
func matchError(err error, lookup errorLookup) (status int, matched error, ok bool) {
	for current := err; current != nil; current = errors.Unwrap(current) {
		if s, found := lookup[reflect.TypeOf(current)]; found {
			return s, current, true
		}
		// Handle multi-errors (errors.Join, etc.)
		if multi, isMulti := current.(interface{ Unwrap() []error }); isMulti {
			for _, inner := range multi.Unwrap() {
				if s, m, found := matchError(inner, lookup); found {
					return s, m, true
				}
			}
		}
	}
	return 0, nil, false
}
