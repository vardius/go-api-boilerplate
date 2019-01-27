/*
Package user holds user domain logic
*/
package user

import (
	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/common/domain"
)

// StreamName for user domain
const StreamName = "user" //fmt.Sprintf("%T", User{})

// User aggregate root
type User struct {
	id      uuid.UUID
	version int
	changes []*domain.Event

	email string
}

func (u *User) transition(e interface{}) {
	switch e := e.(type) {
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

func (u *User) trackChange(e interface{}) error {
	u.transition(e)
	eventEnvelop, err := domain.NewEvent(u.id, StreamName, u.version, e)

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
func (u *User) RegisterWithEmail(id uuid.UUID, email string) error {
	return u.trackChange(&WasRegisteredWithEmail{
		ID:    id,
		Email: email,
	})
}

// RegisterWithGoogle alters current user state and append changes to aggregate root
func (u *User) RegisterWithGoogle(id uuid.UUID, email, googleID string) error {
	return u.trackChange(&WasRegisteredWithGoogle{
		ID:       id,
		Email:    email,
		GoogleID: googleID,
	})
}

// RegisterWithFacebook alters current user state and append changes to aggregate root
func (u *User) RegisterWithFacebook(id uuid.UUID, email, facebookID string) error {
	return u.trackChange(&WasRegisteredWithFacebook{
		ID:         id,
		Email:      email,
		FacebookID: facebookID,
	})
}

// ChangeEmailAddress alters current user state and append changes to aggregate root
func (u *User) ChangeEmailAddress(email string) error {
	return u.trackChange(&EmailAddressWasChanged{
		ID:    u.id,
		Email: email,
	})
}

// FromHistory loads current aggregate root state by applying all events in order
func (u *User) FromHistory(events []*domain.Event) {
	for _, domainEvent := range events {
		u.transition(domainEvent.Payload)
		u.version++
	}
}

// New creates an User
func New() *User {
	return &User{}
}
