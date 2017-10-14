package errors

import "errors"

// ErrEmptyRequestBody is when an request has empty body.
var ErrEmptyRequestBody = errors.New("Empty request body")

// ErrInvalidUrlParams is when an request has invalid or missing parameters.
var ErrInvalidUrlParams = errors.New("Invalid request URL params")
