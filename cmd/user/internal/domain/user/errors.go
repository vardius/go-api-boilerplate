package user

import (
	"fmt"
)

// ErrAlreadyRegistered is when user with given email already exist.
var ErrAlreadyRegistered = fmt.Errorf("user is already registered")
