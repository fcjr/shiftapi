package shiftapi

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// hasHeaderTag returns true if the struct field has a `header` tag.
func hasHeaderTag(f reflect.StructField) bool {
	return f.Tag.Get("header") != ""
}

// headerFieldName returns the header name for a struct field.
func headerFieldName(f reflect.StructField) string {
	name, _, _ := strings.Cut(f.Tag.Get("header"), ",")
	if name == "" {
		return f.Name
	}
	return name
}

// resetHeaderFields zeros out any header-tagged fields on a struct value.
// This is called after body decode so that header-tagged fields are only
// populated by parseHeadersInto, not by JSON keys that happen to match.
func resetHeaderFields(rv reflect.Value) {
	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	rt := rv.Type()
	for i := range rt.NumField() {
		f := rt.Field(i)
		if f.IsExported() && hasHeaderTag(f) {
			rv.Field(i).SetZero()
		}
	}
}

// parseHeadersInto populates header-tagged fields on an existing struct value
// from HTTP headers. Non-header fields are left untouched.
// Only scalar types and pointer-to-scalar types are supported (no slices).
func parseHeadersInto(rv reflect.Value, header http.Header) error {
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		rv = rv.Elem()
	}

	rt := rv.Type()
	if rt.Kind() != reflect.Struct {
		return fmt.Errorf("header type must be a struct, got %s", rt.Kind())
	}

	for i := range rt.NumField() {
		field := rt.Field(i)
		if !field.IsExported() || !hasHeaderTag(field) {
			continue
		}

		name := headerFieldName(field)
		fv := rv.Field(i)
		ft := field.Type

		// Handle pointer fields (optional headers)
		if ft.Kind() == reflect.Pointer {
			raw := header.Get(name)
			if raw == "" {
				continue
			}
			ptr := reflect.New(ft.Elem())
			if err := setScalarValue(ptr.Elem(), raw); err != nil {
				return &headerParseError{Field: name, Err: err}
			}
			fv.Set(ptr)
			continue
		}

		// Handle scalar fields
		raw := header.Get(name)
		if raw == "" {
			continue
		}
		if err := setScalarValue(fv, raw); err != nil {
			return &headerParseError{Field: name, Err: err}
		}
	}

	return nil
}

// headerParseError is returned when a header value cannot be parsed.
type headerParseError struct {
	Field string
	Err   error
}

func (e *headerParseError) Error() string {
	return fmt.Sprintf("invalid header %q: %v", e.Field, e.Err)
}
