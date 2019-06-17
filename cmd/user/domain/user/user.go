/*
Package user holds user domain logic
*/
package user

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

// StreamName for user domain
var StreamName = fmt.Sprintf("%T", User{})

// User aggregate root
type User struct {
	id      uuid.UUID
	version int
	changes []domain.Event

	email string
}

func (u User) transition(e domain.RawEvent) {
	switch e := e.(type) {
	case WasRegisteredWithEmail:
		u.id = e.ID
		u.email = e.Email
	case WasRegisteredWithGoogle:
		u.id = e.ID
		u.email = e.Email
	case WasRegisteredWithFacebook:
		u.id = e.ID
		u.email = e.Email
	case EmailAddressWasChanged:
		u.email = e.Email
	}
}

func (u User) trackChange(e domain.RawEvent) error {
	u.transition(e)
	eventEnvelop, err := domain.NewEvent(u.id, StreamName, u.version, e)

	if err != nil {
		return err
	}

	u.changes = append(u.changes, eventEnvelop)

	return nil
}

// ID returns aggregate root id
func (u User) ID() uuid.UUID {
	return u.id
}

// Version returns current aggregate root version
func (u User) Version() int {
	return u.version
}

// Changes returns all new applied events
func (u User) Changes() []domain.Event {
	return u.changes
}

// RegisterWithEmail alters current user state and append changes to aggregate root
func (u User) RegisterWithEmail(id uuid.UUID, email string) error {
	return u.trackChange(WasRegisteredWithEmail{
		ID:    id,
		Email: email,
	})
}

// RegisterWithGoogle alters current user state and append changes to aggregate root
func (u User) RegisterWithGoogle(id uuid.UUID, email, googleID string) error {
	return u.trackChange(WasRegisteredWithGoogle{
		ID:       id,
		Email:    email,
		GoogleID: googleID,
	})
}

// ConnectWithGoogle alters current user state and append changes to aggregate root
func (u User) ConnectWithGoogle(googleID string) error {
	return u.trackChange(ConnectedWithGoogle{
		ID:       u.id,
		GoogleID: googleID,
	})
}

// RegisterWithFacebook alters current user state and append changes to aggregate root
func (u User) RegisterWithFacebook(id uuid.UUID, email, facebookID string) error {
	return u.trackChange(WasRegisteredWithFacebook{
		ID:         id,
		Email:      email,
		FacebookID: facebookID,
	})
}

// ConnectWithFacebook alters current user state and append changes to aggregate root
func (u User) ConnectWithFacebook(facebookID string) error {
	return u.trackChange(ConnectedWithFacebook{
		ID:         u.id,
		FacebookID: facebookID,
	})
}

// ChangeEmailAddress alters current user state and append changes to aggregate root
func (u User) ChangeEmailAddress(email string) error {
	return u.trackChange(EmailAddressWasChanged{
		ID:    u.id,
		Email: email,
	})
}

// RequestAccessToken dispatches AccessTokenWasRequested event
func (u User) RequestAccessToken() error {
	return u.trackChange(AccessTokenWasRequested{
		ID:    u.id,
		Email: u.email,
	})
}

// FromHistory loads current aggregate root state by applying all events in order
func (u User) FromHistory(events []domain.Event) {
	for _, domainEvent := range events {
		var e domain.RawEvent

		switch domainEvent.Metadata.Type {
		case (AccessTokenWasRequested{}).GetType():
			e = AccessTokenWasRequested{}
		case (EmailAddressWasChanged{}).GetType():
			e = EmailAddressWasChanged{}
		case (WasRegisteredWithEmail{}).GetType():
			e = WasRegisteredWithEmail{}
		case (WasRegisteredWithFacebook{}).GetType():
			e = WasRegisteredWithFacebook{}
		case (ConnectedWithFacebook{}).GetType():
			e = ConnectedWithFacebook{}
		case (WasRegisteredWithGoogle{}).GetType():
			e = WasRegisteredWithGoogle{}
		case (ConnectedWithGoogle{}).GetType():
			e = ConnectedWithGoogle{}
		default:
			// @TODO: should we panic here ?
			log.Panicf("Unhandled user event %s", domainEvent.Metadata.Type)
		}

		err := json.Unmarshal(domainEvent.Payload, &e)
		if err != nil {
			// @TODO: should we panic here ?
			log.Panicf("Error while parsing json to a user event %s, %s", domainEvent.Metadata.Type, domainEvent.Payload)
			continue
		}

		u.transition(e)
		u.version++
	}
}

// New creates an User
func New() User {
	return User{}
}
