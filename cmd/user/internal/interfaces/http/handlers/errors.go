package handlers

import (
	"fmt"
)

// ErrEmptyRequestBody is when an request has empty body.
var ErrEmptyRequestBody = fmt.Errorf("empty request body")

// ErrInvalidURLParams is when an request has invalid or missing parameters.
var ErrInvalidURLParams = fmt.Errorf("invalid request URL parameters")
