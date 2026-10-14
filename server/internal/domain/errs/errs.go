// Package errs defines the error type that domain rules and use cases return.
// Adapters translate its Kind into transport status codes; Code and Message are
// safe to show to clients, while Err (the cause) is only ever logged.
package errs

import (
	"errors"
	"fmt"
)

type Kind int

const (
	KindInvalid Kind = iota + 1
	KindUnauthenticated
	KindForbidden
	KindNotFound
	KindConflict
	KindTooLarge
	KindRateLimited
)

func (k Kind) String() string {
	switch k {
	case KindInvalid:
		return "invalid"
	case KindUnauthenticated:
		return "unauthenticated"
	case KindForbidden:
		return "forbidden"
	case KindNotFound:
		return "not_found"
	case KindConflict:
		return "conflict"
	case KindTooLarge:
		return "too_large"
	case KindRateLimited:
		return "rate_limited"
	default:
		return fmt.Sprintf("kind(%d)", int(k))
	}
}

// FieldError points at one invalid input field.
type FieldError struct {
	Field   string
	Message string
}

type Error struct {
	Kind    Kind
	Code    string // stable snake_case identifier clients can switch on
	Message string // human-readable and safe to return to clients
	Fields  []FieldError
	Err     error // underlying cause, never shown to clients
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Code, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Code)
}

func (e *Error) Unwrap() error { return e.Err }

// Wrap attaches an underlying cause and returns the same error for chaining.
func (e *Error) Wrap(cause error) *Error {
	e.Err = cause
	return e
}

func Invalid(code, message string, fields ...FieldError) *Error {
	return &Error{Kind: KindInvalid, Code: code, Message: message, Fields: fields}
}

func Unauthenticated(code, message string) *Error {
	return &Error{Kind: KindUnauthenticated, Code: code, Message: message}
}

func Forbidden(code, message string) *Error {
	return &Error{Kind: KindForbidden, Code: code, Message: message}
}

func NotFound(code, message string) *Error {
	return &Error{Kind: KindNotFound, Code: code, Message: message}
}

func Conflict(code, message string) *Error {
	return &Error{Kind: KindConflict, Code: code, Message: message}
}

func TooLarge(code, message string) *Error {
	return &Error{Kind: KindTooLarge, Code: code, Message: message}
}

func RateLimited(code, message string) *Error {
	return &Error{Kind: KindRateLimited, Code: code, Message: message}
}

// As returns the domain error in err's chain, if there is one.
func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// IsKind reports whether err's chain holds a domain error of the given kind.
func IsKind(err error, kind Kind) bool {
	e, ok := As(err)
	return ok && e.Kind == kind
}
