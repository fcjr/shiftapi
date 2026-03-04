package shiftapi

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// hasQueryTag returns true if the struct field has a `query` tag.
func hasQueryTag(f reflect.StructField) bool {
	return f.Tag.Get("query") != ""
}

// partitionFields inspects a struct type and reports whether it contains
// query-tagged fields, header-tagged fields, and/or body (json-tagged or
// untagged non-query/non-header) fields.
func partitionFields(t reflect.Type) (hasQuery, hasHeader, hasBody bool) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false, false, false
	}
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if hasQueryTag(f) {
			hasQuery = true
		} else if hasHeaderTag(f) {
			hasHeader = true
		} else {
			// Any exported field without a query or header tag is a body field
			jsonTag := f.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}
			hasBody = true
		}
	}
	return
}

// resetQueryFields zeros out any query-tagged fields on a struct value.
// This is called after body decode so that query-tagged fields are only
// populated by parseQueryInto, not by JSON keys that happen to match.
func resetQueryFields(rv reflect.Value) {
	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	rt := rv.Type()
	for i := range rt.NumField() {
		f := rt.Field(i)
		if f.IsExported() && hasQueryTag(f) {
			rv.Field(i).SetZero()
		}
	}
}

// parseQueryInto populates query-tagged fields on an existing struct value
// from URL query parameters. Non-query fields are left untouched.
func parseQueryInto(rv reflect.Value, values url.Values) error {
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		rv = rv.Elem()
	}

	rt := rv.Type()
	if rt.Kind() != reflect.Struct {
		return fmt.Errorf("query type must be a struct, got %s", rt.Kind())
	}

	for i := range rt.NumField() {
		field := rt.Field(i)
		if !field.IsExported() || !hasQueryTag(field) {
			continue
		}

		name := queryFieldName(field)
		fv := rv.Field(i)
		ft := field.Type

		// Handle pointer fields (optional params)
		if ft.Kind() == reflect.Pointer {
			rawValues, exists := values[name]
			if !exists || len(rawValues) == 0 {
				continue
			}
			ptr := reflect.New(ft.Elem())
			if err := setScalarValue(ptr.Elem(), rawValues[0]); err != nil {
				return &queryParseError{Field: name, Err: err}
			}
			fv.Set(ptr)
			continue
		}

		// Handle slice fields
		if ft.Kind() == reflect.Slice {
			rawValues, exists := values[name]
			if !exists || len(rawValues) == 0 {
				continue
			}
			elemType := ft.Elem()
			slice := reflect.MakeSlice(ft, len(rawValues), len(rawValues))
			for j, raw := range rawValues {
				elem := reflect.New(elemType).Elem()
				if err := setScalarValue(elem, raw); err != nil {
					return &queryParseError{Field: name, Err: err}
				}
				slice.Index(j).Set(elem)
			}
			fv.Set(slice)
			continue
		}

		// Handle scalar fields
		raw := values.Get(name)
		if raw == "" {
			continue
		}
		if err := setScalarValue(fv, raw); err != nil {
			return &queryParseError{Field: name, Err: err}
		}
	}

	return nil
}

// queryFieldName returns the query parameter name for a struct field.
// The field must have a non-empty `query` tag (guaranteed by hasQueryTag).
func queryFieldName(f reflect.StructField) string {
	name, _, _ := strings.Cut(f.Tag.Get("query"), ",")
	if name == "" {
		return f.Name
	}
	return name
}

// setScalarValue parses a string and sets the value on a reflect.Value.
func setScalarValue(v reflect.Value, raw string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(raw)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("invalid boolean value %q", raw)
		}
		v.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid integer value %q", raw)
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value %q", raw)
		}
		v.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(raw, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid float value %q", raw)
		}
		v.SetFloat(n)
	default:
		return fmt.Errorf("unsupported parameter type %s", v.Kind())
	}
	return nil
}

// queryParseError is returned when a query parameter cannot be parsed.
type queryParseError struct {
	Field string
	Err   error
}

func (e *queryParseError) Error() string {
	return fmt.Sprintf("invalid query parameter %q: %v", e.Field, e.Err)
}
