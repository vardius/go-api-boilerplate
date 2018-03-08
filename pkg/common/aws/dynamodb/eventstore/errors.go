package eventstore

import "errors"

// ErrEventNotFound is thrown when an event is not found in the store.
var ErrEventNotFound = errors.New("Event not found")
