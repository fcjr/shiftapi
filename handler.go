package shiftapi

import (
	"encoding/json"
	"errors"
	"net/http"
)

// HandlerFunc is a typed handler for methods without a request body (GET, DELETE, HEAD, etc.).
type HandlerFunc[Resp any] func(r *http.Request) (Resp, error)

// HandlerFuncWithBody is a typed handler for methods with a request body (POST, PUT, PATCH, etc.).
type HandlerFuncWithBody[Body, Resp any] func(r *http.Request, body Body) (Resp, error)

func adapt[Resp any](fn HandlerFunc[Resp], status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := fn(r)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, status, resp)
	}
}

func adaptWithBody[Body, Resp any](fn HandlerFuncWithBody[Body, Resp], status int, validate func(any) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body Body
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, Error(http.StatusBadRequest, "invalid request body"))
			return
		}
		if err := validate(body); err != nil {
			writeError(w, err)
			return
		}
		resp, err := fn(r, body)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, status, resp)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
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
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
