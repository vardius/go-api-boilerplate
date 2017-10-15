package user

import "errors"

// AlreadyRegistered is when user with given email already exist.
var AlreadyRegistered = errors.New("User is already regitered")
