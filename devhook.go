//go:build !shiftapidev

package shiftapi

import "net/http"

func devInit(_ *API)        {}
func devNotifyRoute(_ *API) {}

// ServeHTTP implements http.Handler. Production build — direct passthrough.
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}
