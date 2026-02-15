package shiftapi

import "fmt"

// APIError is an error with an HTTP status code. Return it from handlers
// to control the response status code and message.
type APIError struct {
	Status  int    `json:"-"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%d: %s", e.Status, e.Message)
}

// Error creates a new APIError with the given status code and message.
func Error(status int, message string) *APIError {
	return &APIError{Status: status, Message: message}
}
