package shiftapi

import "github.com/getkin/kin-openapi/openapi3"

// Option configures an API.
type Option func(*API)

// Info describes the API.
type Info struct {
	Title          string
	Summary        string
	Description    string
	TermsOfService string
	Contact        *Contact
	License        *License
	Version        string
}

// Contact describes the API contact information.
type Contact struct {
	Name  string
	URL   string
	Email string
}

// License describes the API license.
type License struct {
	Name       string
	URL        string
	Identifier string
}

// ExternalDocs links to external documentation.
type ExternalDocs struct {
	Description string
	URL         string
}

// WithInfo configures the API metadata.
func WithInfo(info Info) Option {
	return func(api *API) {
		api.spec.Info = &openapi3.Info{
			Title:       info.Title,
			Description: info.Description,
			Version:     info.Version,
		}
		if info.Contact != nil {
			api.spec.Info.Contact = &openapi3.Contact{
				Name:  info.Contact.Name,
				URL:   info.Contact.URL,
				Email: info.Contact.Email,
			}
		}
		if info.License != nil {
			api.spec.Info.License = &openapi3.License{
				Name: info.License.Name,
				URL:  info.License.URL,
			}
		}
	}
}

// WithExternalDocs links to external documentation.
func WithExternalDocs(docs ExternalDocs) Option {
	return func(api *API) {
		api.spec.ExternalDocs = &openapi3.ExternalDocs{
			Description: docs.Description,
			URL:         docs.URL,
		}
	}
}
