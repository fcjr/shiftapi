package shiftapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
)

type Handler interface {
	register(server *ShiftAPI) error

	// unimplementable is a method that should never be called.
	// It is simply used to ensure that the Handler interface can only be implemented
	// internally by the shiftapi package.
	unimplementable()
}

type HandlerOption func(Handler) Handler

type ValidBody any // TODO can we type constrain to a struct?

type HandlerFunc[RequestBody ValidBody, ResponseBody ValidBody] func(
	ctx context.Context,
	headers http.Header,
	requestBody RequestBody,
) (responseBody ResponseBody, err error)

// TODO pass status code
type handler[RequestBody ValidBody, ResponseBody ValidBody] struct {
	method      string
	path        string
	handlerFunc HandlerFunc[RequestBody, ResponseBody]

	info *HandlerInfo
}

// ensure handler implements Handler at compile time
var _ Handler = handler[any, any]{}

func (h handler[RequestBody, ResponseBody]) unimplementable() {
	panic("unimplementable called")
}

func (h handler[RequestBody, ResponseBody]) register(server *ShiftAPI) error {
	var requestBody RequestBody
	inType := reflect.TypeOf(requestBody)
	var responseBody ResponseBody
	outType := reflect.TypeOf(responseBody)

	if err := server.updateSchema(
		h.method,
		h.path,
		inType,
		outType,
		h.info,
	); err != nil {
		return err
	}

	pattern := fmt.Sprintf("%s %s", h.method, h.path)
	stdHandler := h.stdHandler(server.baseContext)
	server.mux.HandleFunc(pattern, stdHandler)
	return nil
}

func (h handler[RequestBody, ResponseBody]) stdHandler(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: valdiate request body?
		var requestBody RequestBody
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		responseBody, err := h.handlerFunc(
			ctx,
			r.Header,
			requestBody,
		)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(responseBody); err != nil {
			http.Error(w, "error encoding response", http.StatusInternalServerError)
			return
		}
	}
}
