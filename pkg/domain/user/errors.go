package user

import "errors"

// UserAlreadyRegistered is when user with given email already exist.
var UserAlreadyRegistered = errors.New("User is already regitered")
