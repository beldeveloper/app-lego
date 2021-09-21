package model

import "errors"

var (
	// ErrNotFound represents the error for the cases when some entity is not found.
	ErrNotFound = errors.New("not found")
	// ErrBadInput represents the error for the cases when the user input is invalid.
	ErrBadInput = errors.New("bad input")
	// ErrBuildCanceled represents the error for the case when the build process was automatically canceled.
	ErrBuildCanceled = errors.New("build canceled")
	// ErrConfigurationNotFound represents the error for the case when the configuration was not found.
	ErrConfigurationNotFound = errors.New("configuration is not found")
)
