// Package errors implements functions to manipulate errors.
package application

import "errors"

// Application errors
var (
	ErrInvalid           = errors.New("validation failed")
	ErrUnauthorized      = errors.New("access denied")
	ErrForbidden         = errors.New("forbidden")
	ErrNotFound          = errors.New("not found")
	ErrInternal          = errors.New("internal system error")
	ErrTemporaryDisabled = errors.New("temporary disabled")
	ErrTimeout           = errors.New("timeout")
)
