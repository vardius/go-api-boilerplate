/*
Package http provides user http client
*/
package http

import "github.com/vardius/go-api-boilerplate/pkg/common/application/errors"

// ErrEmptyRequestBody is when an request has empty body.
var ErrEmptyRequestBody = errors.New(errors.INTERNAL, "Empty request body")

// ErrInvalidURLParams is when an request has invalid or missing parameters.
var ErrInvalidURLParams = errors.New(errors.INTERNAL, "Invalid request URL params")
