package shiftapi

import (
	"context"
	"net/http"
)

// ContextKey is a type-safe key for storing and retrieving values from a
// request's context. The type parameter T determines the type of value
// associated with this key, eliminating the need for type assertions when
// reading values back.
//
// Create keys with [NewContextKey] and use them with [SetContext] and
// [FromContext]:
//
//	var userKey = shiftapi.NewContextKey[User]("user")
//
//	// In middleware:
//	r = shiftapi.SetContext(r, userKey, authenticatedUser)
//
//	// In handler:
//	user, ok := shiftapi.FromContext(r, userKey)
type ContextKey[T any] struct {
	name string
}

// NewContextKey creates a new typed context key. The name is used only for
// debugging; uniqueness is guaranteed by the key's pointer identity, not
// its name.
func NewContextKey[T any](name string) *ContextKey[T] {
	return &ContextKey[T]{name: name}
}

// String returns the key's name for debugging purposes.
func (k *ContextKey[T]) String() string {
	return k.name
}

// SetContext returns a shallow copy of r with the given typed value stored
// in its context. Use this in middleware to pass data to downstream handlers:
//
//	func authMiddleware(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        user, err := authenticate(r)
//	        if err != nil {
//	            http.Error(w, "unauthorized", http.StatusUnauthorized)
//	            return
//	        }
//	        next.ServeHTTP(w, shiftapi.SetContext(r, userKey, user))
//	    })
//	}
func SetContext[T any](r *http.Request, key *ContextKey[T], val T) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), key, val))
}

// FromContext retrieves a typed value from the request's context. The second
// return value reports whether the key was present.
//
//	user, ok := shiftapi.FromContext(r, userKey)
func FromContext[T any](r *http.Request, key *ContextKey[T]) (T, bool) {
	val, ok := r.Context().Value(key).(T)
	return val, ok
}
