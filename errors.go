package shiftapi

import "fmt"

// APIError is an error with an HTTP status code. Return it from handlers
// to control the response status code and message. Unrecognized errors
// (i.e. errors that are not [*APIError] or [*ValidationError]) are mapped
// to 500 Internal Server Error to prevent leaking implementation details.
type APIError struct {
	Status  int    `json:"-"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%d: %s", e.Status, e.Message)
}

// Error creates a new [APIError] with the given HTTP status code and message.
//
//	return nil, shiftapi.Error(http.StatusNotFound, "user not found")
func Error(status int, message string) *APIError {
	return &APIError{Status: status, Message: message}
}
