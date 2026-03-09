//go:build shiftapidev

package shiftapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"
)

// devInit is called from New(). It sets up development-mode features gated
// behind the shiftapidev build tag.
//
// Normal dev mode: binds a random free port and prints it to stderr for the
// Vite/Next.js plugin. The listener is started later by devServe (via
// sync.Once in ServeHTTP) to guarantee all routes are registered first.
//
// Export mode (SHIFTAPI_EXPORT_SPEC=1): sets exportMode so that each route
// registration writes the current spec JSON to a temp file (via
// devNotifyRoute). A background goroutine waits for stability and calls
// os.Exit(0). Even if the main goroutine exits first via log.Fatal (when
// the user's port is unavailable), the temp file already contains the
// complete spec — it was written synchronously during route registration.
func devInit(api *API) {
	if os.Getenv("SHIFTAPI_EXPORT_SPEC") != "" {
		api.exportMode = true
		api.exportFile = os.Getenv("SHIFTAPI_EXPORT_FILE")
		if api.exportFile == "" {
			fmt.Fprintf(os.Stderr, "shiftapi: SHIFTAPI_EXPORT_FILE is required in export mode\n")
			os.Exit(1)
		}
		go exportExitWhenReady(api)
		return
	}

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("shiftapi dev: failed to bind dev port: %v", err)
	}
	api.devListener = ln
	port := ln.Addr().(*net.TCPAddr).Port
	fmt.Fprintf(os.Stderr, "shiftapi:dev:port=%d\n", port)
	log.Printf("shiftapi: running in dev mode (shiftapidev build tag), dev port %d", port)
}

// ServeHTTP implements http.Handler. Dev build — on the first call, starts
// the dev listener via sync.Once before forwarding to the mux.
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.devOnce.Do(func() {
		if a.devListener != nil {
			go http.Serve(a.devListener, a.mux)
		}
	})
	a.mux.ServeHTTP(w, r)
}

// devNotifyRoute is called after each route registration. In export mode it
// writes the current spec JSON to the temp file. This runs synchronously on
// the main goroutine, ensuring the file is always up-to-date before the
// main goroutine can reach http.ListenAndServe / log.Fatal.
func devNotifyRoute(api *API) {
	if !api.exportMode {
		return
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(api.spec); err != nil {
		return
	}
	_ = os.WriteFile(api.exportFile, buf.Bytes(), 0644)
}

// exportExitWhenReady waits for routes to stabilise, then exits cleanly.
// The spec file is already written by devNotifyRoute — this goroutine's
// only job is to terminate the process when the user's port is free
// (http.ListenAndServe would otherwise block forever).
func exportExitWhenReady(api *API) {
	// Wait for at least one route.
	for api.spec.Paths == nil || api.spec.Paths.Len() == 0 {
		runtime.Gosched()
	}
	// Wait for the count to be stable for 5ms — generous enough to span
	// any gap between route registrations (~20–100μs each).
	prev := api.spec.Paths.Len()
	for {
		time.Sleep(5 * time.Millisecond)
		cur := api.spec.Paths.Len()
		if cur == prev {
			break
		}
		prev = cur
	}
	os.Exit(0)
}
