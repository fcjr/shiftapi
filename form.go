package shiftapi

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"reflect"
	"slices"
	"strings"
)

var (
	fileHeaderType      = reflect.TypeFor[*multipart.FileHeader]()
	fileHeaderSliceType = reflect.TypeFor[[]*multipart.FileHeader]()
)

// hasFormTag returns true if the struct field has a `form` tag.
func hasFormTag(f reflect.StructField) bool {
	return f.Tag.Get("form") != ""
}

// formFieldName returns the form field name from the struct tag.
func formFieldName(f reflect.StructField) string {
	name, _, _ := strings.Cut(f.Tag.Get("form"), ",")
	if name == "" {
		return f.Name
	}
	return name
}

// isFileField returns true if the field type is *multipart.FileHeader or []*multipart.FileHeader.
func isFileField(f reflect.StructField) bool {
	return f.Type == fileHeaderType || f.Type == fileHeaderSliceType
}

// acceptTypes returns the accepted MIME types from the `accept` struct tag.
// Returns nil if no accept tag is present.
func acceptTypes(f reflect.StructField) []string {
	tag := f.Tag.Get("accept")
	if tag == "" {
		return nil
	}
	var types []string
	for part := range strings.SplitSeq(tag, ",") {
		t := strings.TrimSpace(part)
		if t != "" {
			types = append(types, t)
		}
	}
	return types
}

// checkFileContentType validates the Content-Type of an uploaded file
// against the accepted types. Returns an error if the type is not allowed.
func checkFileContentType(fh *multipart.FileHeader, name string, allowed []string) error {
	if len(allowed) == 0 {
		return nil
	}
	ct := fh.Header.Get("Content-Type")
	if slices.Contains(allowed, ct) {
		return nil
	}
	return &formParseError{
		Field: name,
		Err:   fmt.Errorf("content type %q not allowed, accepted: %s", ct, strings.Join(allowed, ", ")),
	}
}

// parseFormInto parses a multipart form request into struct fields tagged with `form`.
func parseFormInto(rv reflect.Value, r *http.Request, maxMemory int64) error {
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		return &formParseError{Err: fmt.Errorf("failed to parse multipart form: %w", err)}
	}

	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		rv = rv.Elem()
	}

	rt := rv.Type()
	if rt.Kind() != reflect.Struct {
		return fmt.Errorf("form type must be a struct, got %s", rt.Kind())
	}

	for i := range rt.NumField() {
		field := rt.Field(i)
		if !field.IsExported() || !hasFormTag(field) {
			continue
		}

		name := formFieldName(field)
		fv := rv.Field(i)

		if field.Type == fileHeaderType {
			// Single file: *multipart.FileHeader
			_, fh, err := r.FormFile(name)
			if err != nil {
				if err == http.ErrMissingFile {
					continue
				}
				return &formParseError{Field: name, Err: err}
			}
			if allowed := acceptTypes(field); allowed != nil {
				if err := checkFileContentType(fh, name, allowed); err != nil {
					return err
				}
			}
			fv.Set(reflect.ValueOf(fh))
			continue
		}

		if field.Type == fileHeaderSliceType {
			// Multiple files: []*multipart.FileHeader
			if r.MultipartForm != nil && r.MultipartForm.File != nil {
				files := r.MultipartForm.File[name]
				if allowed := acceptTypes(field); allowed != nil {
					for _, fh := range files {
						if err := checkFileContentType(fh, name, allowed); err != nil {
							return err
						}
					}
				}
				if len(files) > 0 {
					fv.Set(reflect.ValueOf(files))
				}
			}
			continue
		}

		// Text form field — use r.FormValue and setScalarValue
		raw := r.FormValue(name)
		if raw == "" {
			continue
		}
		if err := setScalarValue(fv, raw); err != nil {
			return &formParseError{Field: name, Err: err}
		}
	}

	return nil
}

// formParseError is returned when a form field cannot be parsed.
type formParseError struct {
	Field string
	Err   error
}

func (e *formParseError) Error() string {
	if e.Field == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("invalid form field %q: %v", e.Field, e.Err)
}
