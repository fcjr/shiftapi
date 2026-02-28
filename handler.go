package shiftapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"reflect"
)

// HandlerFunc is a typed handler for routes.
// The In struct's fields are discriminated by struct tags:
// fields with `query:"..."` tags are parsed from query parameters,
// fields with `header:"..."` tags are parsed from HTTP headers,
// and remaining fields (with `json:"..."` tags or untagged) are parsed from the request body.
// For routes without input, use struct{} as the In type.
type HandlerFunc[In, Resp any] func(r *http.Request, in In) (Resp, error)

func adapt[In, Resp any](fn HandlerFunc[In, Resp], status int, validate func(any) error, hasQuery, hasHeader, hasBody bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in In
		rv := reflect.ValueOf(&in).Elem()

		// JSON-decode body if there are body fields
		if hasBody {
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				writeError(w, Error(http.StatusBadRequest, "invalid request body"))
				return
			}
			// Re-point rv after decode (in case In is a pointer that was nil)
			rv = reflect.ValueOf(&in).Elem()
		}

		// Reset any query-tagged fields that body decode may have
		// inadvertently set, so they only come from URL query params.
		if hasBody && hasQuery {
			resetQueryFields(rv)
		}

		// Reset any header-tagged fields that body decode may have
		// inadvertently set, so they only come from HTTP headers.
		if hasBody && hasHeader {
			resetHeaderFields(rv)
		}

		// Parse query params if there are query fields
		if hasQuery {
			if err := parseQueryInto(rv, r.URL.Query()); err != nil {
				writeError(w, Error(http.StatusBadRequest, err.Error()))
				return
			}
		}

		// Parse headers if there are header fields
		if hasHeader {
			if err := parseHeadersInto(rv, r.Header); err != nil {
				writeError(w, Error(http.StatusBadRequest, err.Error()))
				return
			}
		}

		if err := validate(in); err != nil {
			writeError(w, err)
			return
		}

		resp, err := fn(r, in)
		if err != nil {
			writeError(w, err)
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

func writeError(w http.ResponseWriter, err error) {
	var valErr *ValidationError
	if errors.As(err, &valErr) {
		writeJSON(w, http.StatusUnprocessableEntity, valErr)
		return
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		writeJSON(w, apiErr.Status, apiErr)
		return
	}
	writeJSON(w, http.StatusInternalServerError, &APIError{Status: http.StatusInternalServerError, Message: "internal server error"})
}
