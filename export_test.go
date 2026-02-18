package shiftapi

import "github.com/getkin/kin-openapi/openapi3"

// Spec is exported only for testing.
func (a *API) Spec() *openapi3.T {
	return a.spec
}
