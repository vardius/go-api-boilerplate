/*
Package user holds user domain logic
*/
package user

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
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

// New creates an User
func New() User {
	return User{}
}

// FromHistory loads current aggregate root state by applying all events in order
func FromHistory(events []domain.Event) User {
	u := New()

	for _, domainEvent := range events {
		var e domain.RawEvent

		switch domainEvent.Metadata.Type {
		case (AccessTokenWasRequested{}).GetType():
			accessTokenWasRequested := AccessTokenWasRequested{}
			err := unmarshalPayload(domainEvent.Payload, &accessTokenWasRequested)
			if err != nil {
				log.Panicf("Error while trying to unmarshal user event %s. %s", domainEvent.Metadata.Type, err)
			}

			e = accessTokenWasRequested
		case (EmailAddressWasChanged{}).GetType():
			emailAddressWasChanged := EmailAddressWasChanged{}
			err := unmarshalPayload(domainEvent.Payload, &emailAddressWasChanged)
			if err != nil {
				log.Panicf("Error while trying to unmarshal user event %s. %s", domainEvent.Metadata.Type, err)
			}

			e = emailAddressWasChanged
		case (WasRegisteredWithEmail{}).GetType():
			wasRegisteredWithEmail := WasRegisteredWithEmail{}
			err := unmarshalPayload(domainEvent.Payload, &wasRegisteredWithEmail)
			if err != nil {
				log.Panicf("Error while trying to unmarshal user event %s. %s", domainEvent.Metadata.Type, err)
			}

			e = wasRegisteredWithEmail
		case (WasRegisteredWithFacebook{}).GetType():
			wasRegisteredWithFacebook := WasRegisteredWithFacebook{}
			err := unmarshalPayload(domainEvent.Payload, &wasRegisteredWithFacebook)
			if err != nil {
				log.Panicf("Error while trying to unmarshal user event %s. %s", domainEvent.Metadata.Type, err)
			}

			e = wasRegisteredWithFacebook
		case (ConnectedWithFacebook{}).GetType():
			connectedWithFacebook := ConnectedWithFacebook{}
			err := unmarshalPayload(domainEvent.Payload, &connectedWithFacebook)
			if err != nil {
				log.Panicf("Error while trying to unmarshal user event %s. %s", domainEvent.Metadata.Type, err)
			}

			e = connectedWithFacebook
		case (WasRegisteredWithGoogle{}).GetType():
			wasRegisteredWithGoogle := WasRegisteredWithGoogle{}
			err := unmarshalPayload(domainEvent.Payload, &wasRegisteredWithGoogle)
			if err != nil {
				log.Panicf("Error while trying to unmarshal user event %s. %s", domainEvent.Metadata.Type, err)
			}

			e = wasRegisteredWithGoogle
		case (ConnectedWithGoogle{}).GetType():
			connectedWithGoogle := ConnectedWithGoogle{}
			err := unmarshalPayload(domainEvent.Payload, &connectedWithGoogle)
			if err != nil {
				log.Panicf("Error while trying to unmarshal user event %s. %s", domainEvent.Metadata.Type, err)
			}

			e = connectedWithGoogle
		default:
			log.Panicf("Unhandled user event %s", domainEvent.Metadata.Type)
		}

		u.transition(e)
		u.version++
	}

	return u
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
func (u *User) RegisterWithEmail(id uuid.UUID, email string) error {
	return u.trackChange(WasRegisteredWithEmail{
		ID:    id,
		Email: email,
	})
}

// RegisterWithGoogle alters current user state and append changes to aggregate root
func (u *User) RegisterWithGoogle(id uuid.UUID, email, googleID string) error {
	return u.trackChange(WasRegisteredWithGoogle{
		ID:       id,
		Email:    email,
		GoogleID: googleID,
	})
}

// ConnectWithGoogle alters current user state and append changes to aggregate root
func (u *User) ConnectWithGoogle(googleID string) error {
	return u.trackChange(ConnectedWithGoogle{
		ID:       u.id,
		GoogleID: googleID,
	})
}

// RegisterWithFacebook alters current user state and append changes to aggregate root
func (u *User) RegisterWithFacebook(id uuid.UUID, email, facebookID string) error {
	return u.trackChange(WasRegisteredWithFacebook{
		ID:         id,
		Email:      email,
		FacebookID: facebookID,
	})
}

// ConnectWithFacebook alters current user state and append changes to aggregate root
func (u *User) ConnectWithFacebook(facebookID string) error {
	return u.trackChange(ConnectedWithFacebook{
		ID:         u.id,
		FacebookID: facebookID,
	})
}

// ChangeEmailAddress alters current user state and append changes to aggregate root
func (u *User) ChangeEmailAddress(email string) error {
	return u.trackChange(EmailAddressWasChanged{
		ID:    u.id,
		Email: email,
	})
}

// RequestAccessToken dispatches AccessTokenWasRequested event
func (u *User) RequestAccessToken() error {
	return u.trackChange(AccessTokenWasRequested{
		ID:    u.id,
		Email: u.email,
	})
}

func (u *User) trackChange(e domain.RawEvent) error {
	u.transition(e)
	event, err := domain.NewEvent(u.id, StreamName, u.version, e)

	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "User trackChange error")
	}

	u.changes = append(u.changes, event)

	return nil
}

func (u *User) transition(e domain.RawEvent) {
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
