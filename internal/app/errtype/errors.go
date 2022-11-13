package errtype

import "errors"

var (
	// ErrNotFound represents the error for the cases when some entity is not found.
	ErrNotFound = errors.New("not found")
	// ErrBadInput represents the error for the cases when the user input is invalid.
	ErrBadInput = errors.New("bad input")
	// ErrUnauthorized represents the error for the cases when the authorization is required.
	ErrUnauthorized = errors.New("unauthorized")
)
