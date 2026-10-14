// Package page holds cursor-pagination types shared by every list use case.
package page

import (
	"encoding/base64"
	"encoding/json"

	"passwordmanage/server/internal/domain/errs"
)

const (
	DefaultLimit = 50
	MaxLimit     = 200
)

// Request is a validated page request. Cursor is opaque to clients.
type Request struct {
	Limit  int
	Cursor string
}

// Result is one page of items plus the cursor for the next page ("" when done).
type Result[T any] struct {
	Items      []T
	NextCursor string
}

// NewRequest validates client input; a nil limit means DefaultLimit.
func NewRequest(limit *int, cursor *string) (Request, error) {
	req := Request{Limit: DefaultLimit}
	if limit != nil {
		if *limit < 1 || *limit > MaxLimit {
			return Request{}, errs.Invalid("invalid_input", "Invalid page limit.",
				errs.FieldError{Field: "limit", Message: "must be between 1 and 200"})
		}
		req.Limit = *limit
	}
	if cursor != nil {
		req.Cursor = *cursor
	}
	return req, nil
}

// EncodeCursor turns a keyset position into an opaque, URL-safe cursor.
func EncodeCursor[T any](position T) (string, error) {
	raw, err := json.Marshal(position)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeCursor reverses EncodeCursor; a tampered cursor is a client error.
func DecodeCursor[T any](cursor string) (T, error) {
	var position T
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err == nil {
		err = json.Unmarshal(raw, &position)
	}
	if err != nil {
		return position, errs.Invalid("invalid_input", "Invalid page cursor.",
			errs.FieldError{Field: "cursor", Message: "is not a valid cursor"}).Wrap(err)
	}
	return position, nil
}
