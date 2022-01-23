/*
Package user holds user domain logic
*/
package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"

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
	changes []*domain.Event

	email EmailAddress
}

// New creates an User
func New() User {
	return User{}
}

// FromHistory loads current aggregate root state by applying all events in order
func FromHistory(ctx context.Context, events []*domain.Event) (User, error) {
	u := New()

	for _, domainEvent := range events {
		var e domain.RawEvent

		switch domainEvent.Type {
		case AccessTokenWasRequestedType:
			e = domainEvent.Payload.(*AccessTokenWasRequested)
		case EmailAddressWasChangedType:
			e = domainEvent.Payload.(*EmailAddressWasChanged)
		case WasRegisteredWithEmailType:
			e = domainEvent.Payload.(*WasRegisteredWithEmail)
		case WasRegisteredWithFacebookType:
			e = domainEvent.Payload.(*WasRegisteredWithFacebook)
		case ConnectedWithFacebookType:
			e = domainEvent.Payload.(*ConnectedWithFacebook)
		case WasRegisteredWithGoogleType:
			e = domainEvent.Payload.(*WasRegisteredWithGoogle)
		case ConnectedWithGoogleType:
			e = domainEvent.Payload.(*ConnectedWithGoogle)
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
func (u User) Changes() []*domain.Event {
	return u.changes
}

// RegisterWithEmail alters current user state and append changes to aggregate root
func (u *User) RegisterWithEmail(ctx context.Context, id uuid.UUID, email EmailAddress) error {
	e := &WasRegisteredWithEmail{
		ID:    id,
		Email: email,
	}

	if err := u.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	domainEvent, err := domain.NewEventFromRawEvent(u.id, StreamName, u.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if err := u.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

// RegisterWithGoogle alters current user state and append changes to aggregate root
func (u *User) RegisterWithGoogle(ctx context.Context, id uuid.UUID, email EmailAddress, googleID, accessToken, redirectPath string) error {
	e := &WasRegisteredWithGoogle{
		ID:           id,
		Email:        email,
		GoogleID:     googleID,
		AccessToken:  accessToken,
		RedirectPath: redirectPath,
	}

	if err := u.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	domainEvent, err := domain.NewEventFromRawEvent(u.id, StreamName, u.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if err := u.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

// ConnectWithGoogle alters current user state and append changes to aggregate root
func (u *User) ConnectWithGoogle(ctx context.Context, googleID, accessToken, redirectPath string) error {
	e := &ConnectedWithGoogle{
		ID:           u.id,
		GoogleID:     googleID,
		AccessToken:  accessToken,
		RedirectPath: redirectPath,
	}

	if err := u.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	domainEvent, err := domain.NewEventFromRawEvent(u.id, StreamName, u.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if err := u.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

// RegisterWithFacebook alters current user state and append changes to aggregate root
func (u *User) RegisterWithFacebook(ctx context.Context, id uuid.UUID, email EmailAddress, facebookID, accessToken, redirectPath string) error {
	e := &WasRegisteredWithFacebook{
		ID:           id,
		Email:        email,
		FacebookID:   facebookID,
		AccessToken:  accessToken,
		RedirectPath: redirectPath,
	}

	if err := u.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	domainEvent, err := domain.NewEventFromRawEvent(u.id, StreamName, u.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if err := u.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

// ConnectWithFacebook alters current user state and append changes to aggregate root
func (u *User) ConnectWithFacebook(ctx context.Context, facebookID, accessToken, redirectPath string) error {
	e := &ConnectedWithFacebook{
		ID:           u.id,
		FacebookID:   facebookID,
		AccessToken:  accessToken,
		RedirectPath: redirectPath,
	}

	if err := u.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	domainEvent, err := domain.NewEventFromRawEvent(u.id, StreamName, u.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if err := u.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

// ChangeEmailAddress alters current user state and append changes to aggregate root
func (u *User) ChangeEmailAddress(ctx context.Context, email EmailAddress) error {
	e := &EmailAddressWasChanged{
		ID:    u.id,
		Email: email,
	}

	if err := u.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	domainEvent, err := domain.NewEventFromRawEvent(u.id, StreamName, u.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if err := u.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

// RequestAccessToken dispatches AccessTokenWasRequested event
func (u *User) RequestAccessToken(ctx context.Context, redirectPath string) error {
	e := &AccessTokenWasRequested{
		ID:           u.id,
		Email:        u.email,
		RedirectPath: redirectPath,
	}

	if err := u.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	domainEvent, err := domain.NewEventFromRawEvent(u.id, StreamName, u.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if err := u.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

func (u *User) trackChange(ctx context.Context, event *domain.Event) error {
	var meta domain.EventMetadata
	if i, hasIdentity := identity.FromContext(ctx); hasIdentity {
		meta.Identity = i
	}
	if m, ok := metadata.FromContext(ctx); ok {
		meta.IPAddress = m.IPAddress
		meta.UserAgent = m.UserAgent
		meta.Referer = m.Referer
	}
	if !meta.IsEmpty() {
		event.WithMetadata(&meta)
	}

	u.changes = append(u.changes, event)
	u.version++

	return nil
}

func (u *User) transition(e domain.RawEvent) error {
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

	return nil
}
