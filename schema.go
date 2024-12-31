package shiftapi

import (
	"github.com/pb33f/libopenapi/datamodel/high/base"
)

type Info struct {
	Summary        string
	Title          string
	Description    string
	TermsOfService string
	Contact        *Contact
	License        *License
	Version        string
}

type Contact struct {
	Name  string
	URL   string
	Email string
}

type License struct {
	Name       string
	URL        string
	Identifier string
}

func WithInfo(info Info) func(*ShiftAPI) *ShiftAPI {
	return func(api *ShiftAPI) *ShiftAPI {
		api.spec.Info = &base.Info{
			Title:       info.Title,
			Description: info.Description,
			Version:     info.Version,
		}
		if info.Contact != nil {
			api.spec.Info.Contact = &base.Contact{
				Name:  info.Contact.Name,
				URL:   info.Contact.URL,
				Email: info.Contact.Email,
			}
		}
		if info.License != nil {
			api.spec.Info.License = &base.License{
				Name: info.License.Name,
				URL:  info.License.URL,
			}
		}
		return api
	}
}

type ExternalDocs struct {
	Description string
	URL         string
}

func WithExternalDocs(externalDocs ExternalDocs) func(*ShiftAPI) *ShiftAPI {
	return func(api *ShiftAPI) *ShiftAPI {
		api.spec.ExternalDocs = &base.ExternalDoc{
			Description: externalDocs.Description,
			URL:         externalDocs.URL,
		}
		return api
	}
}
