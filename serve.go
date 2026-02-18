//go:build !shiftapidev

package shiftapi

import "net/http"

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
	return http.ListenAndServe(addr, api)
}
