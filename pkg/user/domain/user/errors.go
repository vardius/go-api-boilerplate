package user

import "github.com/vardius/go-api-boilerplate/pkg/common/application/errors"

// ErrAlreadyRegistered is when user with given email already exist.
var ErrAlreadyRegistered = errors.New(errors.INTERNAL, "User is already registered")
