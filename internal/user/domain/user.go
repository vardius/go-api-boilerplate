/*
Package user holds user domain logic and aggregate roots
*/
package user

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

// User aggregate root
type User struct {
	id      uuid.UUID
	version int
	changes []*domain.Event

	email string
}

func (u *User) transition(event interface{}) {
	switch e := event.(type) {
	case *WasRegisteredWithEmail:
		u.id = e.ID
		u.email = e.Email
	case *WasRegisteredWithGoogle:
		u.id = e.ID
		u.email = e.Email
	case *WasRegisteredWithFacebook:
		u.id = e.ID
		u.email = e.Email
	case *EmailAddressWasChanged:
		u.email = e.Email
	}
}

func (u *User) trackChange(event interface{}) error {
	u.transition(event)
	eventEnvelop, err := domain.NewEvent(u.id, fmt.Sprintf("%T", u), u.version, event)

	if err != nil {
		return err
	}

	u.changes = append(u.changes, eventEnvelop)

	return nil
}

// ID returns aggregate root id
func (u *User) ID() uuid.UUID {
	return u.id
}

// Version returns current aggregate root version
func (u *User) Version() int {
	return u.version
}

// Changes returns all new applied events
func (u *User) Changes() []*domain.Event {
	return u.changes
}

// RegisterWithEmail alters current user state and append changes to aggregate root
func (u *User) RegisterWithEmail(id uuid.UUID, email string, authToken string) error {
	return u.trackChange(&WasRegisteredWithEmail{id, email, authToken})
}

// RegisterWithGoogle alters current user state and append changes to aggregate root
func (u *User) RegisterWithGoogle(id uuid.UUID, email string, authToken string) error {
	return u.trackChange(&WasRegisteredWithGoogle{id, email, authToken})
}

// RegisterWithFacebook alters current user state and append changes to aggregate root
func (u *User) RegisterWithFacebook(id uuid.UUID, email string, authToken string) error {
	return u.trackChange(&WasRegisteredWithFacebook{id, email, authToken})
}

// ChangeEmailAddress alters current user state and append changes to aggregate root
func (u *User) ChangeEmailAddress(email string) error {
	return u.trackChange(&EmailAddressWasChanged{u.id, email})
}

// FromHistory loads current aggregate root state by applying all events in order
func (u *User) FromHistory(events []*domain.Event) {
	for _, event := range events {
		u.transition(event.Payload)
		u.version++
	}
}

// New creates an User
func New() *User {
	return &User{}
}
