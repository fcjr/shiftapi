package shiftapi

import (
	"reflect"
)

// Scalar is a constraint that permits any Go type whose underlying type is a
// string, integer, or float — the kinds commonly used as enum carriers.
type Scalar interface {
	~string |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// WithEnum registers the given values as the complete set of allowed enum
// values for type T. When T appears as a struct field in a request or response
// type, the generated OpenAPI schema will include an enum constraint with these
// values — no validate:"oneof=..." tag required.
//
// If a field also carries a validate:"oneof=..." tag, that tag takes precedence
// over the registered values.
//
//	type Status string
//
//	const (
//		StatusActive   Status = "active"
//		StatusInactive Status = "inactive"
//		StatusPending  Status = "pending"
//	)
//
//	api := shiftapi.New(
//		shiftapi.WithEnum[Status](StatusActive, StatusInactive, StatusPending),
//	)
func WithEnum[T Scalar](values ...T) apiOptionFunc {
	return func(api *API) {
		t := reflect.TypeFor[T]()
		enums := make([]any, len(values))
		for i, v := range values {
			enums[i] = v
		}
		api.enumRegistry[t] = enums
	}
}

// lookupEnum returns the registered enum values for the given type, or nil if
// none have been registered. Pointer types are dereferenced before lookup.
func (a *API) lookupEnum(t reflect.Type) []any {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if v, ok := a.enumRegistry[t]; ok {
		return v
	}
	return nil
}
