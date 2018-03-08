package user

import "errors"

// ErrAlreadyRegistered is when user with given email already exist.
var ErrAlreadyRegistered = errors.New("User is already regitered")
