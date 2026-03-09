package shiftapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"reflect"
)

// RawHandlerFunc is a handler function that writes directly to the
// [http.ResponseWriter]. Unlike [HandlerFunc] it has only one type parameter
// for the input — the handler owns the response lifecycle entirely, which
// makes it suitable for streaming (SSE), file downloads, WebSocket upgrades,
// and other use cases where JSON encoding is inappropriate.
//
// The input struct In is parsed and validated identically to [HandlerFunc]:
// path, query, header, json, and form tags all work as expected. For
// POST/PUT/PATCH methods the body is decoded only when the input struct
// contains json or form-tagged fields, leaving r.Body available otherwise.
type RawHandlerFunc[In any] func(w http.ResponseWriter, r *http.Request, in In) error

// writeTracker wraps an http.ResponseWriter and records whether Write or
// WriteHeader has been called. It implements Unwrap() so that callers can
// reach the underlying ResponseWriter for http.Flusher, http.Hijacker, etc.
type writeTracker struct {
	http.ResponseWriter
	written bool
}

func (wt *writeTracker) WriteHeader(code int) {
	wt.written = true
	wt.ResponseWriter.WriteHeader(code)
}

func (wt *writeTracker) Write(b []byte) (int, error) {
	wt.written = true
	return wt.ResponseWriter.Write(b)
}

// Unwrap returns the underlying ResponseWriter so that callers can type-assert
// to http.Flusher, http.Hijacker, etc.
func (wt *writeTracker) Unwrap() http.ResponseWriter {
	return wt.ResponseWriter
}

// HandlerFunc is a typed handler function for API routes. The type parameters
// In and Resp are the request and response types — both are automatically
// reflected into the OpenAPI schema.
//
// The In struct's fields are discriminated by struct tags:
//   - path:"name" — parsed from URL path parameters (e.g. /users/{id})
//   - query:"name" — parsed from URL query parameters
//   - header:"name" — parsed from HTTP request headers
//   - json:"name" — parsed from the JSON request body (default for POST/PUT/PATCH)
//   - form:"name" — parsed from multipart/form-data (for file uploads)
//
// The Resp struct's fields may also use the header tag to set response headers:
//   - header:"name" — written as an HTTP response header (excluded from JSON body)
//
// Header-tagged fields on the response are automatically stripped from the JSON
// body and documented as response headers in the OpenAPI spec. Use a pointer
// field (e.g. *string) for optional response headers that may not always be set.
//
// Use struct{} as In for routes that take no input, or as Resp for routes
// that return no body (e.g. health checks that only need a status code).
//
// The [*http.Request] parameter gives access to cookies, path
// parameters, and other request metadata.
type HandlerFunc[In, Resp any] func(r *http.Request, in In) (Resp, error)

// isNoBodyStatus reports whether the HTTP status code forbids a response body.
// Per RFC 9110, only 204 No Content and 304 Not Modified must not contain a body.
func isNoBodyStatus(status int) bool {
	return status == http.StatusNoContent || status == http.StatusNotModified
}

func adapt[In, Resp any](fn HandlerFunc[In, Resp], status int, validate func(any) error, hasPath, hasQuery, hasHeader, hasBody, hasForm bool, noBody bool, respEnc *respEncoder, staticHeaders []staticResponseHeader, maxUploadSize int64, errLookup errorLookup, badRequestFn, internalServerFn func(error) any) http.HandlerFunc {
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

			// Reset any query/path/header-tagged fields that body decode may have
			// inadvertently set, so they only come from their proper source.
			if hasQuery {
				resetQueryFields(rv)
			}
			if hasPath {
				resetPathFields(rv)
			}
			if hasHeader {
				resetHeaderFields(rv)
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

		// Parse headers if there are header fields
		if hasHeader {
			if err := parseHeadersInto(rv, r.Header); err != nil {
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
		for _, h := range staticHeaders {
			w.Header().Set(h.name, h.value)
		}
		if respEnc != nil {
			writeResponseHeaders(w, resp)
		}
		if noBody {
			w.WriteHeader(status)
			return
		}
		if respEnc != nil {
			writeJSON(w, status, respEnc.encode(resp))
			return
		}
		writeJSON(w, status, resp)
	}
}

func adaptRaw[In any](fn RawHandlerFunc[In], validate func(any) error, hasPath, hasQuery, hasHeader, hasBody, hasForm bool, staticHeaders []staticResponseHeader, maxUploadSize int64, errLookup errorLookup, badRequestFn, internalServerFn func(error) any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in In
		rv := reflect.ValueOf(&in).Elem()

		if hasForm {
			if err := parseFormInto(rv, r, maxUploadSize); err != nil {
				writeJSON(w, http.StatusBadRequest, badRequestFn(err))
				return
			}
			rv = reflect.ValueOf(&in).Elem()
		} else if hasBody {
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				writeJSON(w, http.StatusBadRequest, badRequestFn(err))
				return
			}
			rv = reflect.ValueOf(&in).Elem()
			if hasQuery {
				resetQueryFields(rv)
			}
			if hasPath {
				resetPathFields(rv)
			}
			if hasHeader {
				resetHeaderFields(rv)
			}
		}

		if hasQuery {
			if err := parseQueryInto(rv, r.URL.Query()); err != nil {
				writeJSON(w, http.StatusBadRequest, badRequestFn(err))
				return
			}
		}
		if hasPath {
			if err := parsePathInto(rv, r); err != nil {
				writeJSON(w, http.StatusBadRequest, badRequestFn(err))
				return
			}
		}
		if hasHeader {
			if err := parseHeadersInto(rv, r.Header); err != nil {
				writeJSON(w, http.StatusBadRequest, badRequestFn(err))
				return
			}
		}

		if err := validate(in); err != nil {
			handleError(w, internalServerFn, err, errLookup)
			return
		}

		for _, h := range staticHeaders {
			w.Header().Set(h.name, h.value)
		}

		wt := &writeTracker{ResponseWriter: w}
		if err := fn(wt, r, in); err != nil {
			if !wt.written {
				handleError(w, internalServerFn, err, errLookup)
			} else {
				log.Printf("shiftapi: raw handler error after response started: %v", err)
			}
		}
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
