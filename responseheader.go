package shiftapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
)

var jsonMarshalerType = reflect.TypeFor[json.Marshaler]()

// hasRespHeaderFields reports whether the type has any exported fields with a `header` tag.
// This is used for response types to determine if response headers need to be written.
func hasRespHeaderFields(t reflect.Type) bool {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return false
	}
	for f := range t.Fields() {
		if f.IsExported() && hasHeaderTag(f) {
			return true
		}
	}
	return false
}

// writeResponseHeaders extracts header-tagged fields from a response value
// and sets them on the ResponseWriter. Fields are formatted as their string
// representation. Only scalar types and pointer-to-scalar types are supported.
func writeResponseHeaders(w http.ResponseWriter, resp any) {
	rv := reflect.ValueOf(resp)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}

	rt := rv.Type()
	for i := range rt.NumField() {
		field := rt.Field(i)
		if !field.IsExported() || !hasHeaderTag(field) {
			continue
		}

		name := http.CanonicalHeaderKey(headerFieldName(field))
		fv := rv.Field(i)

		// Pointer fields are optional — skip when nil.
		if fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		}

		// Non-pointer fields are always sent (consistent with required
		// semantics elsewhere in the framework).
		w.Header().Set(name, fmt.Sprint(fv.Interface()))
	}
}

// respEncoder strips header-tagged fields from a response before JSON encoding.
// The derived struct type is built once at registration time so that runtime
// encoding uses a real struct — preserving omitempty, custom marshalers on
// field types, embedded structs, and all other encoding/json behavior.
//
// When the response type itself implements json.Marshaler, the encoder falls
// back to encoding the original value (customJSON mode) so the custom method
// is preserved. Header fields may appear in the JSON body in this case — the
// user controls their own encoding.
type respEncoder struct {
	derivedType  reflect.Type // struct type without header fields (nil when customJSON)
	fieldMapping []int        // derivedType field i ← original field fieldMapping[i]
	customJSON   bool         // true when the original type implements json.Marshaler
}

// newRespEncoder builds a derived struct type from t that excludes all
// header-tagged fields. Returns nil if t is not a struct.
//
// If t (or *t) implements json.Marshaler, the encoder is returned with
// customJSON set to true — the original value is encoded as-is so the
// custom MarshalJSON is preserved, and only header extraction is performed.
// Unexported fields are excluded from the derived type — they don't affect
// JSON output and cannot be copied via reflect.
func newRespEncoder(t reflect.Type) *respEncoder {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	// If the type has a custom MarshalJSON, we can't use a derived struct
	// (it would lose the method). Fall back to encoding the original value.
	if t.Implements(jsonMarshalerType) || reflect.PointerTo(t).Implements(jsonMarshalerType) {
		return &respEncoder{customJSON: true}
	}

	var fields []reflect.StructField
	var mapping []int
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue // skip unexported — can't copy via reflect, invisible to encoding/json
		}
		if hasHeaderTag(f) {
			continue
		}
		fields = append(fields, f)
		mapping = append(mapping, i)
	}

	return &respEncoder{
		derivedType:  reflect.StructOf(fields),
		fieldMapping: mapping,
	}
}

// encode returns a value suitable for JSON encoding. When the response type
// has a custom MarshalJSON, it returns the original value unchanged. Otherwise
// it copies non-header fields into the derived struct type.
func (e *respEncoder) encode(resp any) any {
	if e.customJSON {
		return resp
	}

	rv := reflect.ValueOf(resp)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return resp
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return resp
	}

	derived := reflect.New(e.derivedType).Elem()
	for i, origIdx := range e.fieldMapping {
		derived.Field(i).Set(rv.Field(origIdx))
	}
	return derived.Interface()
}

// generateRespHeaders builds OpenAPI header definitions from header-tagged
// fields on a response struct type.
func generateRespHeaders(t reflect.Type) openapi3.Headers {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	headers := make(openapi3.Headers)
	for f := range t.Fields() {
		if !f.IsExported() || !hasHeaderTag(f) {
			continue
		}

		name := http.CanonicalHeaderKey(headerFieldName(f))
		schema := scalarToOpenAPISchema(f.Type)

		required := f.Type.Kind() != reflect.Pointer
		headers[name] = &openapi3.HeaderRef{
			Value: &openapi3.Header{
				Parameter: openapi3.Parameter{
					Name:     name,
					In:       "header",
					Required: required,
					Schema:   schema,
				},
			},
		}
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}

// stripRespHeaderFields removes header-tagged fields from a response body
// schema's Properties and Required slices so they don't appear in the JSON body.
func stripRespHeaderFields(t reflect.Type, schema *openapi3.Schema) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || schema == nil {
		return
	}
	for f := range t.Fields() {
		if !f.IsExported() || !hasHeaderTag(f) {
			continue
		}
		jname := jsonFieldName(f)
		if jname == "" || jname == "-" {
			continue
		}
		delete(schema.Properties, jname)
		for j, req := range schema.Required {
			if req == jname {
				schema.Required = append(schema.Required[:j], schema.Required[j+1:]...)
				break
			}
		}
	}
}
