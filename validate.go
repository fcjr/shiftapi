package shiftapi

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-playground/validator/v10"
)

// ValidationError is returned when request body validation fails.
type ValidationError struct {
	Message string       `json:"message"`
	Errors  []FieldError `json:"errors"`
}

func (e *ValidationError) Error() string {
	return e.Message
}

// FieldError describes a single field validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// WithValidator sets a custom validator instance on the API.
func WithValidator(v *validator.Validate) Option {
	return func(api *API) {
		api.validate = v
	}
}

// validateStruct validates a struct value using the provided validator.
// It dereferences pointers and skips non-struct types.
func validateStruct(v *validator.Validate, val any) error {
	rv := reflect.ValueOf(val)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	if err := v.Struct(rv.Interface()); err != nil {
		var ve validator.ValidationErrors
		if ok := errors.As(err, &ve); ok {
			fieldErrors := make([]FieldError, len(ve))
			for i, fe := range ve {
				fieldErrors[i] = FieldError{
					Field:   fe.Field(),
					Message: fieldErrorMessage(fe),
				}
			}
			return &ValidationError{
				Message: "validation failed",
				Errors:  fieldErrors,
			}
		}
		return err
	}
	return nil
}

// fieldErrorMessage returns a human-readable message for a field validation error.
func fieldErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "email":
		return "must be a valid email address"
	case "url", "uri":
		return "must be a valid URI"
	case "uuid", "uuid3", "uuid4", "uuid5":
		return "must be a valid UUID"
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fe.Param())
	case "gt":
		return fmt.Sprintf("must be greater than %s", fe.Param())
	case "lt":
		return fmt.Sprintf("must be less than %s", fe.Param())
	case "len":
		return fmt.Sprintf("must have length %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	case "datetime":
		return "must be a valid date-time"
	default:
		return fmt.Sprintf("failed %s validation", fe.Tag())
	}
}

// validateSchemaCustomizer is a SchemaCustomizerFn that reads validate tags
// and maps them to OpenAPI schema properties.
func validateSchemaCustomizer(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
	validateTag := tag.Get("validate")
	if validateTag == "" {
		return nil
	}

	rules := strings.SplitSeq(validateTag, ",")
	for rule := range rules {
		rule = strings.TrimSpace(rule)

		key, param, _ := strings.Cut(rule, "=")

		switch key {
		case "email":
			schema.Format = "email"
		case "url", "uri":
			schema.Format = "uri"
		case "uuid", "uuid3", "uuid4", "uuid5":
			schema.Format = "uuid"
		case "datetime":
			schema.Format = "date-time"
		case "min":
			applyMin(t, schema, param)
		case "max":
			applyMax(t, schema, param)
		case "gte":
			if n, err := strconv.ParseFloat(param, 64); err == nil {
				schema.Min = &n
			}
		case "lte":
			if n, err := strconv.ParseFloat(param, 64); err == nil {
				schema.Max = &n
			}
		case "gt":
			if n, err := strconv.ParseFloat(param, 64); err == nil {
				schema.Min = &n
				schema.ExclusiveMin = true
			}
		case "lt":
			if n, err := strconv.ParseFloat(param, 64); err == nil {
				schema.Max = &n
				schema.ExclusiveMax = true
			}
		case "len":
			applyLen(t, schema, param)
		case "oneof":
			values := strings.Fields(param)
			enums := make([]any, len(values))
			for i, v := range values {
				enums[i] = v
			}
			schema.Enum = enums
		}
	}
	return nil
}

// applyMin sets minLength/minimum/minItems depending on the field's underlying kind.
func applyMin(t reflect.Type, schema *openapi3.Schema, param string) {
	kind := derefKind(t)
	switch {
	case isStringKind(kind):
		if n, err := strconv.ParseUint(param, 10, 64); err == nil {
			schema.MinLength = n
		}
	case isNumericKind(kind):
		if n, err := strconv.ParseFloat(param, 64); err == nil {
			schema.Min = &n
		}
	case isSliceKind(kind):
		if n, err := strconv.ParseUint(param, 10, 64); err == nil {
			schema.MinItems = n
		}
	}
}

// applyMax sets maxLength/maximum/maxItems depending on the field's underlying kind.
func applyMax(t reflect.Type, schema *openapi3.Schema, param string) {
	kind := derefKind(t)
	switch {
	case isStringKind(kind):
		if n, err := strconv.ParseUint(param, 10, 64); err == nil {
			v := &n
			schema.MaxLength = v
		}
	case isNumericKind(kind):
		if n, err := strconv.ParseFloat(param, 64); err == nil {
			schema.Max = &n
		}
	case isSliceKind(kind):
		if n, err := strconv.ParseUint(param, 10, 64); err == nil {
			v := &n
			schema.MaxItems = v
		}
	}
}

// applyLen sets both min and max length/items to the same value.
func applyLen(t reflect.Type, schema *openapi3.Schema, param string) {
	kind := derefKind(t)
	switch {
	case isStringKind(kind):
		if n, err := strconv.ParseUint(param, 10, 64); err == nil {
			schema.MinLength = n
			schema.MaxLength = &n
		}
	case isSliceKind(kind):
		if n, err := strconv.ParseUint(param, 10, 64); err == nil {
			schema.MinItems = n
			schema.MaxItems = &n
		}
	}
}

func derefKind(t reflect.Type) reflect.Kind {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Kind()
}

func isStringKind(k reflect.Kind) bool {
	return k == reflect.String
}

func isNumericKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func isSliceKind(k reflect.Kind) bool {
	return k == reflect.Slice || k == reflect.Array
}

// applyRequired walks struct fields and adds JSON names to schema.Required
// for fields that have validate:"required".
func applyRequired(t reflect.Type, schema *openapi3.Schema) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}

	for i := range t.NumField() {
		field := t.Field(i)
		validateTag := field.Tag.Get("validate")
		if !hasRule(validateTag, "required") {
			continue
		}
		jsonName := jsonFieldName(field)
		if jsonName == "" || jsonName == "-" {
			continue
		}
		schema.Required = append(schema.Required, jsonName)
	}
}

// hasRule checks whether a comma-separated validate tag contains the given rule name.
func hasRule(tag, rule string) bool {
	for r := range strings.SplitSeq(tag, ",") {
		key, _, _ := strings.Cut(strings.TrimSpace(r), "=")
		if key == rule {
			return true
		}
	}
	return false
}

// jsonFieldName returns the JSON field name for a struct field.
func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		return f.Name
	}
	return name
}
