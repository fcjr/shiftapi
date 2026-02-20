package shiftapi

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// parseQuery populates a struct of type T from URL query parameters.
// It uses the `query` struct tag for parameter names, falling back to the field name.
// A tag of `query:"-"` causes the field to be skipped.
func parseQuery[T any](values url.Values) (T, error) {
	var result T
	rv := reflect.ValueOf(&result).Elem()
	rt := rv.Type()

	for rt.Kind() == reflect.Ptr {
		rv.Set(reflect.New(rt.Elem()))
		rv = rv.Elem()
		rt = rt.Elem()
	}

	if rt.Kind() != reflect.Struct {
		return result, fmt.Errorf("query type must be a struct, got %s", rt.Kind())
	}

	for i := range rt.NumField() {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}

		name := queryFieldName(field)
		if name == "-" {
			continue
		}

		fv := rv.Field(i)
		ft := field.Type

		// Handle pointer fields (optional params)
		if ft.Kind() == reflect.Ptr {
			rawValues, exists := values[name]
			if !exists || len(rawValues) == 0 {
				continue // leave nil
			}
			ptr := reflect.New(ft.Elem())
			if err := setScalarValue(ptr.Elem(), rawValues[0]); err != nil {
				return result, &queryParseError{Field: name, Err: err}
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
					return result, &queryParseError{Field: name, Err: err}
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
			return result, &queryParseError{Field: name, Err: err}
		}
	}

	return result, nil
}

// queryFieldName returns the query parameter name for a struct field.
func queryFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("query")
	if tag == "" {
		return f.Name
	}
	name, _, _ := strings.Cut(tag, ",")
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
		return fmt.Errorf("unsupported query parameter type %s", v.Kind())
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
