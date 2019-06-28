package eventstore

import "github.com/vardius/go-api-boilerplate/pkg/errors"

// ErrEventNotFound is thrown when an event is not found in the store.
var ErrEventNotFound = errors.New(errors.NOTFOUND, "Event not found")
