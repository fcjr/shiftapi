package shiftapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

const defaultReadTimeout = 10 * time.Second
const defaultWriteTimeout = 10 * time.Second
const defaultShutdownGracePeriod = 10 * time.Second

type ShiftAPI struct {
	spec    *openapi3.T
	specGen *openapi3gen.Generator
	mux     *http.ServeMux
}

func New(
	options ...func(*ShiftAPI) *ShiftAPI,
) *ShiftAPI {
	mux := http.NewServeMux()
	spec := &openapi3.T{
		OpenAPI: "3.1",
		Paths:   &openapi3.Paths{},
		Components: &openapi3.Components{
			Schemas: make(openapi3.Schemas),
		},
	}

	api := &ShiftAPI{
		spec:    spec,
		specGen: openapi3gen.NewGenerator(),
		mux:     mux,
	}
	for _, option := range options {
		api = option(api)
	}
	mux.HandleFunc("GET /openapi.json", api.serveSchema)
	mux.HandleFunc("GET /docs", api.serveDocs)
	mux.HandleFunc("GET /", api.redirectTo("/docs"))
	return api
}

// Register adds 1 or more handlers to the server.
// The handlers are expected to be created via the shiftapi.Post, shiftapi.Get,
// shiftapi.Put, shiftapi.Patch, and shiftapi.Delete functions.
func (s *ShiftAPI) Register(handlers ...Handler) error {
	for _, h := range handlers {
		if err := h.register(s); err != nil {
			return err
		}
	}
	return nil
}

func (s *ShiftAPI) redirectTo(path string) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		http.Redirect(res, req, path, http.StatusTemporaryRedirect)
	}
}

func (s *ShiftAPI) serveSchema(res http.ResponseWriter, req *http.Request) {
	x, err := s.spec.MarshalYAML()
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	b, err := json.MarshalIndent(x, "", "  ")
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	_, _ = res.Write(b)
}

func (s *ShiftAPI) serveDocs(res http.ResponseWriter, req *http.Request) {
	title := ""
	if s.spec.Info != nil {
		title = s.spec.Info.Title
	}
	err := genRedocHTML(redocData{
		Title:   title,
		SpecURL: "/openapi.json",
	}, res)
	if err != nil {
		fmt.Println(err)
		res.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *ShiftAPI) ListenAndServe(ctx context.Context, addr string) error {
	// TODO add address to schema & create server separately http server

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownGracePeriod)
		defer cancel()
		return httpServer.Shutdown(ctx)
	case err := <-errCh:
		return err
	}
}
