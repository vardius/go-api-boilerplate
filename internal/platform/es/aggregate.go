package es

import (
	"slices"
	"uuid"
)

// Aggregate is a consistency boundary rebuilt from its own stream. Embed
// Changes to implement it:
//
//	type User struct {
//		es.Changes
//		id    uuid.UUID
//		email Email
//	}
//
//	func (u *User) ID() uuid.UUID { return u.id }
//
//	func (u *User) Apply(event any) error {
//		switch e := event.(type) {
//		case Registered:
//			u.id, u.email = e.UserID, e.Email
//		default:
//			return fmt.Errorf("user: can't apply %T", event)
//		}
//		return nil
//	}
type Aggregate interface {
	// ID is the ID of the aggregate's stream.
	ID() uuid.UUID
	// Apply changes the aggregate's state to reflect one event. It only
	// assigns: rules are checked before an event is raised, and stored
	// events are facts. It returns an error for an event it doesn't know.
	Apply(event any) error

	changes() *Changes
}

// Changes tracks an aggregate's stored version and the events it raised
// since it was loaded.
type Changes struct {
	version int
	pending []any
}

//lint:ignore U1000 aggregates outside es get it by embedding Changes, which staticcheck can't see from here.
func (c *Changes) changes() *Changes { return c }

// Version is the version of the last stored event the aggregate was built
// from, 0 for a new one.
func (c *Changes) Version() int { return c.version }

// Raise applies event to a and records it, to be stored by the next Save.
func Raise(a Aggregate, event any) error {
	if err := a.Apply(event); err != nil {
		return err
	}
	c := a.changes()
	c.pending = append(c.pending, event)
	return nil
}

// Replay rebuilds a from stored events, as Repository.Load does. Tests use
// it to set up an aggregate's history.
func Replay(a Aggregate, events ...any) error {
	for _, e := range events {
		if err := a.Apply(e); err != nil {
			return err
		}
	}
	a.changes().version += len(events)
	return nil
}

// Pending returns the events a raised that aren't stored yet.
func Pending(a Aggregate) []any {
	return slices.Clone(a.changes().pending)
}
