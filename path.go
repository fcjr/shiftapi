package shiftapi

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// hasPathTag returns true if the struct field has a `path` tag.
func hasPathTag(f reflect.StructField) bool {
	return f.Tag.Get("path") != ""
}

// pathFieldName returns the path parameter name for a struct field.
func pathFieldName(f reflect.StructField) string {
	name, _, _ := strings.Cut(f.Tag.Get("path"), ",")
	if name == "" {
		return f.Name
	}
	return name
}

// resetPathFields zeros out any path-tagged fields on a struct value.
// This is called after body decode so that path-tagged fields are only
// populated by parsePathInto, not by JSON keys that happen to match.
func resetPathFields(rv reflect.Value) {
	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	rt := rv.Type()
	for i := range rt.NumField() {
		f := rt.Field(i)
		if f.IsExported() && hasPathTag(f) {
			rv.Field(i).SetZero()
		}
	}
}

// parsePathInto populates path-tagged fields on an existing struct value
// from URL path parameters via r.PathValue. Only scalar types are supported;
// pointers and slices are rejected at registration time.
func parsePathInto(rv reflect.Value, r *http.Request) error {
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		rv = rv.Elem()
	}

	rt := rv.Type()
	if rt.Kind() != reflect.Struct {
		return fmt.Errorf("path type must be a struct, got %s", rt.Kind())
	}

	for i := range rt.NumField() {
		field := rt.Field(i)
		if !field.IsExported() || !hasPathTag(field) {
			continue
		}

		name := pathFieldName(field)
		raw := r.PathValue(name)
		if raw == "" {
			continue
		}
		if err := setScalarValue(rv.Field(i), raw); err != nil {
			return &pathParseError{Field: name, Err: err}
		}
	}

	return nil
}

// validatePathFields checks that path-tagged fields are scalar types (no
// pointers or slices) and that every path-tagged field name appears in the
// route pattern. Called at registration time.
func validatePathFields(t reflect.Type, routeParams map[string]bool) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	for f := range t.Fields() {
		if !f.IsExported() || !hasPathTag(f) {
			continue
		}
		ft := f.Type
		if ft.Kind() == reflect.Pointer || ft.Kind() == reflect.Slice {
			panic(fmt.Sprintf("shiftapi: path-tagged field %q must be a scalar type, got %s", f.Name, ft.Kind()))
		}
		name := pathFieldName(f)
		if !routeParams[name] {
			panic(fmt.Sprintf("shiftapi: path-tagged field %q not found in route pattern", name))
		}
	}
}

// pathParseError is returned when a path parameter cannot be parsed.
type pathParseError struct {
	Field string
	Err   error
}

func (e *pathParseError) Error() string {
	return fmt.Sprintf("invalid path parameter %q: %v", e.Field, e.Err)
}

func (e *pathParseError) Unwrap() error { return e.Err }
