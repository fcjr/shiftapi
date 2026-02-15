package shiftapi

import (
	"encoding/json"
	"net/http"
	"os"
)

// ListenAndServe starts the HTTP server on the given address.
//
// If the SHIFTAPI_EXPORT_SPEC environment variable is set to a file path,
// the OpenAPI spec is written to that path and the process exits immediately
// without starting the server. This enables build tools (like the Vite plugin)
// to extract the spec by running the Go binary.
func ListenAndServe(addr string, api *API) error {
	if specPath := os.Getenv("SHIFTAPI_EXPORT_SPEC"); specPath != "" {
		if err := exportSpec(api, specPath); err != nil {
			return err
		}
		os.Exit(0)
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
		f.Close()
		return err
	}

	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}

	return f.Close()
}
