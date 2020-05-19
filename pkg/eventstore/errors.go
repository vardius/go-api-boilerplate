package eventstore

import (
	"fmt"
)

// ErrEventNotFound is thrown when an event is not found in the store.
var ErrEventNotFound = fmt.Errorf("event not found")
