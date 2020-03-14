package handlers

import "github.com/vardius/go-api-boilerplate/pkg/errors"

// ErrEmptyRequestBody is when an request has empty body.
var ErrEmptyRequestBody = errors.New(errors.INTERNAL, "Empty request body")

// ErrInvalidURLParams is when an request has invalid or missing parameters.
var ErrInvalidURLParams = errors.New(errors.INTERNAL, "Invalid request URL parameters")
