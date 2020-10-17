/*
Package user holds user domain logic
*/
package user

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	userdomain "github.com/vardius/go-api-boilerplate/cmd/user/internal/domain"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
	"github.com/vardius/go-api-boilerplate/pkg/metadata"
)

// StreamName for user domain
var StreamName = fmt.Sprintf("%T", User{})

// User aggregate root
type User struct {
	id      uuid.UUID
	version int
	changes []domain.Event

	email EmailAddress
}

// New creates an User
func New() User {
	return User{}
}

// FromHistory loads current aggregate root state by applying all events in order
func FromHistory(ctx context.Context, events []domain.Event) (User, error) {
	u := New()

	for _, domainEvent := range events {
		var e domain.RawEvent

		switch domainEvent.Type {
		case (AccessTokenWasRequested{}).GetType():
			accessTokenWasRequested := AccessTokenWasRequested{}
			if err := json.Unmarshal(domainEvent.Payload, &accessTokenWasRequested); err != nil {
				return u, apperrors.Wrap(err)
			}

			e = accessTokenWasRequested
		case (EmailAddressWasChanged{}).GetType():
			emailAddressWasChanged := EmailAddressWasChanged{}
			if err := json.Unmarshal(domainEvent.Payload, &emailAddressWasChanged); err != nil {
				return u, apperrors.Wrap(err)
			}

			e = emailAddressWasChanged
		case (WasRegisteredWithEmail{}).GetType():
			wasRegisteredWithEmail := WasRegisteredWithEmail{}
			if err := json.Unmarshal(domainEvent.Payload, &wasRegisteredWithEmail); err != nil {
				return u, apperrors.Wrap(err)
			}

			e = wasRegisteredWithEmail
		case (WasRegisteredWithFacebook{}).GetType():
			wasRegisteredWithFacebook := WasRegisteredWithFacebook{}
			if err := json.Unmarshal(domainEvent.Payload, &wasRegisteredWithFacebook); err != nil {
				return u, apperrors.Wrap(err)
			}

			e = wasRegisteredWithFacebook
		case (ConnectedWithFacebook{}).GetType():
			connectedWithFacebook := ConnectedWithFacebook{}
			if err := json.Unmarshal(domainEvent.Payload, &connectedWithFacebook); err != nil {
				return u, apperrors.Wrap(err)
			}

			e = connectedWithFacebook
		case (WasRegisteredWithGoogle{}).GetType():
			wasRegisteredWithGoogle := WasRegisteredWithGoogle{}
			if err := json.Unmarshal(domainEvent.Payload, &wasRegisteredWithGoogle); err != nil {
				return u, apperrors.Wrap(err)
			}

			e = wasRegisteredWithGoogle
		case (ConnectedWithGoogle{}).GetType():
			connectedWithGoogle := ConnectedWithGoogle{}
			if err := json.Unmarshal(domainEvent.Payload, &connectedWithGoogle); err != nil {
				return u, apperrors.Wrap(err)
			}

			e = connectedWithGoogle
		default:
			return u, apperrors.Wrap(fmt.Errorf("unhandled user event %s", domainEvent.Type))
		}

		if err := u.transition(e); err != nil {
			return u, apperrors.Wrap(err)
		}

		u.version++
	}

	return u, nil
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
func (u *User) RegisterWithEmail(ctx context.Context, id uuid.UUID, email EmailAddress) error {
	e := WasRegisteredWithEmail{
		ID:    id,
		Email: email,
	}

	domainEvent, err := domain.NewEventFromRawEvent(u.id, StreamName, u.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if err := u.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	if _, err := u.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

// RegisterWithGoogle alters current user state and append changes to aggregate root
func (u *User) RegisterWithGoogle(ctx context.Context, id uuid.UUID, email EmailAddress, googleID, accessToken string) error {
	e := WasRegisteredWithGoogle{
		ID:          id,
		Email:       email,
		GoogleID:    googleID,
		AccessToken: accessToken,
	}

	domainEvent, err := domain.NewEventFromRawEvent(u.id, StreamName, u.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if err := u.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	if _, err := u.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

// ConnectWithGoogle alters current user state and append changes to aggregate root
func (u *User) ConnectWithGoogle(ctx context.Context, googleID, accessToken string) error {
	e := ConnectedWithGoogle{
		ID:          u.id,
		GoogleID:    googleID,
		AccessToken: accessToken,
	}

	domainEvent, err := domain.NewEventFromRawEvent(u.id, StreamName, u.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if err := u.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	if _, err := u.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

// RegisterWithFacebook alters current user state and append changes to aggregate root
func (u *User) RegisterWithFacebook(ctx context.Context, id uuid.UUID, email EmailAddress, facebookID, accessToken string) error {
	e := WasRegisteredWithFacebook{
		ID:          id,
		Email:       email,
		FacebookID:  facebookID,
		AccessToken: accessToken,
	}

	domainEvent, err := domain.NewEventFromRawEvent(u.id, StreamName, u.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if err := u.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	if _, err := u.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

// ConnectWithFacebook alters current user state and append changes to aggregate root
func (u *User) ConnectWithFacebook(ctx context.Context, facebookID, accessToken string) error {
	e := ConnectedWithFacebook{
		ID:          u.id,
		FacebookID:  facebookID,
		AccessToken: accessToken,
	}

	domainEvent, err := domain.NewEventFromRawEvent(u.id, StreamName, u.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if err := u.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	if _, err := u.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

// ChangeEmailAddress alters current user state and append changes to aggregate root
func (u *User) ChangeEmailAddress(ctx context.Context, email EmailAddress) error {
	e := EmailAddressWasChanged{
		ID:    u.id,
		Email: email,
	}

	domainEvent, err := domain.NewEventFromRawEvent(u.id, StreamName, u.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if err := u.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	if _, err := u.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

// RequestAccessToken dispatches AccessTokenWasRequested event
func (u *User) RequestAccessToken(ctx context.Context) error {
	e := AccessTokenWasRequested{
		ID:    u.id,
		Email: u.email,
	}

	domainEvent, err := domain.NewEventFromRawEvent(u.id, StreamName, u.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if err := u.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	if _, err := u.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

func (u *User) trackChange(ctx context.Context, event domain.Event) (domain.Event, error) {
	meta := userdomain.EventMetadata{}
	if i, hasIdentity := identity.FromContext(ctx); hasIdentity {
		meta.Identity = i
	}
	if m, ok := metadata.FromContext(ctx); ok {
		meta.IPAddress = m.IPAddress
		meta.UserAgent = m.UserAgent
		meta.Referer = m.Referer
	}
	if !meta.IsEmpty() {
		if err := event.WithMetadata(meta); err != nil {
			return event, apperrors.Wrap(err)
		}
	}

	u.changes = append(u.changes, event)
	u.version++

	return event, nil
}

func (u *User) transition(e domain.RawEvent) error {
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

	return nil
}
