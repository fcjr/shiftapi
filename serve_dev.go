//go:build shiftapidev

package shiftapi

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// ListenAndServe starts the HTTP server on the given address.
//
// In production builds this is a direct call to [http.ListenAndServe] with
// zero additional overhead.
//
// When built with -tags shiftapidev (used automatically by the Vite plugin),
// the following environment variables are supported:
//   - SHIFTAPI_EXPORT_SPEC=<path>: write the OpenAPI spec to the given file
//     and exit without starting the server.
//   - SHIFTAPI_PORT=<port>: override the port in addr, allowing the Vite
//     plugin to automatically assign a free port.
func ListenAndServe(addr string, api *API) error {
	log.Println("shiftapi: running in dev mode (shiftapidev build tag)")
	if specPath := os.Getenv("SHIFTAPI_EXPORT_SPEC"); specPath != "" {
		if err := exportSpec(api, specPath); err != nil {
			return err
		}
		os.Exit(0)
	}
	if port := os.Getenv("SHIFTAPI_PORT"); port != "" {
		addr = ":" + port
		log.Printf("shiftapi: listening on %s (via SHIFTAPI_PORT)", addr)
	}
	return http.ListenAndServe(addr, api)
}

func exportSpec(api *API, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(api.Spec()); err != nil {
		_ = f.Close()
		return err
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}

	return f.Close()
}
