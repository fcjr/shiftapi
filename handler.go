package shiftapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"reflect"

	"github.com/coder/websocket"
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

// handlerConfig holds the registration-time configuration that adapt and
// adaptRaw capture in their closures. It replaces the long positional
// parameter lists that were previously passed to parseInput, adapt, and
// adaptRaw.
type handlerConfig struct {
	hasPath      bool
	hasQuery     bool
	hasHeader    bool
	decodeBody   bool
	hasForm      bool
	maxUploadSize int64
	staticHeaders []staticResponseHeader
	errLookup     errorLookup
	validate      func(any) error
	badRequestFn  func(error) any
	internalServerFn func(error) any
}

// parseInput decodes and validates the typed input from the request. It returns
// the parsed value and true on success. On failure it writes an error response
// and returns the zero value and false.
func parseInput[In any](w http.ResponseWriter, r *http.Request, hc *handlerConfig) (In, bool) {
	var in In
	rv := reflect.ValueOf(&in).Elem()

	if hc.hasForm {
		if err := parseFormInto(rv, r, hc.maxUploadSize); err != nil {
			writeJSON(w, http.StatusBadRequest, hc.badRequestFn(err))
			return in, false
		}
		rv = reflect.ValueOf(&in).Elem()
	} else if hc.decodeBody {
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, hc.badRequestFn(err))
			return in, false
		}
		rv = reflect.ValueOf(&in).Elem()
		if hc.hasQuery {
			resetQueryFields(rv)
		}
		if hc.hasPath {
			resetPathFields(rv)
		}
		if hc.hasHeader {
			resetHeaderFields(rv)
		}
	}

	if hc.hasQuery {
		if err := parseQueryInto(rv, r.URL.Query()); err != nil {
			writeJSON(w, http.StatusBadRequest, hc.badRequestFn(err))
			return in, false
		}
	}
	if hc.hasPath {
		if err := parsePathInto(rv, r); err != nil {
			writeJSON(w, http.StatusBadRequest, hc.badRequestFn(err))
			return in, false
		}
	}
	if hc.hasHeader {
		if err := parseHeadersInto(rv, r.Header); err != nil {
			writeJSON(w, http.StatusBadRequest, hc.badRequestFn(err))
			return in, false
		}
	}

	if err := hc.validate(in); err != nil {
		handleError(w, hc.internalServerFn, err, hc.errLookup)
		return in, false
	}
	return in, true
}

func adapt[In, Resp any](fn HandlerFunc[In, Resp], hc *handlerConfig, status int, noBody bool, respEnc *respEncoder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in, ok := parseInput[In](w, r, hc)
		if !ok {
			return
		}

		resp, err := fn(r, in)
		if err != nil {
			handleError(w, hc.internalServerFn, err, hc.errLookup)
			return
		}
		for _, h := range hc.staticHeaders {
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

func adaptRaw[In any](fn RawHandlerFunc[In], hc *handlerConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in, ok := parseInput[In](w, r, hc)
		if !ok {
			return
		}

		for _, h := range hc.staticHeaders {
			w.Header().Set(h.name, h.value)
		}

		wt := &writeTracker{ResponseWriter: w}
		if err := fn(wt, r, in); err != nil {
			if !wt.written {
				handleError(wt, hc.internalServerFn, err, hc.errLookup)
			} else {
				log.Printf("shiftapi: raw handler error after response started: %v", err)
			}
		}
	}
}

func adaptSSE[In, Event any](fn SSEHandlerFunc[In, Event], hc *handlerConfig, sendVariants map[reflect.Type]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in, ok := parseInput[In](w, r, hc)
		if !ok {
			return
		}

		for _, h := range hc.staticHeaders {
			w.Header().Set(h.name, h.value)
		}

		wt := &writeTracker{ResponseWriter: w}
		sse := &SSEWriter[Event]{
			w:            wt,
			rc:           http.NewResponseController(wt),
			sendVariants: sendVariants,
		}
		if err := fn(r, in, sse); err != nil {
			if !wt.written {
				handleError(wt, hc.internalServerFn, err, hc.errLookup)
			} else {
				log.Printf("shiftapi: SSE handler error after response started: %v", err)
			}
		}
	}
}

func adaptWSMessages[In any](
	dispatch map[string]wsOnHandler,
	sendVariants map[reflect.Type]string,
	hc *handlerConfig,
	wsOpts *WSAcceptOptions,
	cb wsCallbacks,
	setup func(r *http.Request, ws *WSSender, in In) (any, error),
) http.HandlerFunc {
	// Convert our public WSAcceptOptions to the underlying library's AcceptOptions.
	var acceptOpts *websocket.AcceptOptions
	if wsOpts != nil {
		acceptOpts = &websocket.AcceptOptions{
			Subprotocols:   wsOpts.Subprotocols,
			OriginPatterns: wsOpts.OriginPatterns,
		}
	}

	// In dev mode, skip origin verification so that cross-origin requests
	// from Vite/Next.js dev servers work without extra config. User-provided
	// options (e.g. Subprotocols) are preserved.
	if devMode {
		if acceptOpts == nil {
			acceptOpts = &websocket.AcceptOptions{InsecureSkipVerify: true}
		} else {
			acceptOpts.InsecureSkipVerify = true
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		in, ok := parseInput[In](w, r, hc)
		if !ok {
			return
		}

		conn, err := websocket.Accept(w, r, acceptOpts)
		if err != nil {
			// Accept writes its own error response (e.g. 403 for origin
			// violations), so we must not write a second one.
			return
		}

		ws := &WSSender{conn: conn, ctx: r.Context(), sendVariants: sendVariants}

		state, err := setup(r, ws, in)
		if err != nil {
			log.Printf("shiftapi: WS setup error: %v", err)
			_ = conn.Close(websocket.StatusInternalError, "setup error")
			return
		}

		runWSDispatchLoop(r, conn, ws, state, dispatch, cb)
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
