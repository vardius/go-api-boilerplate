/*
Package token holds token domain logic
*/
package token

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gopkg.in/oauth2.v4"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
	"github.com/vardius/go-api-boilerplate/pkg/metadata"
)

// StreamName for token domain
var StreamName = fmt.Sprintf("%T", Token{})

// Token aggregate root
type Token struct {
	id      uuid.UUID
	userID  uuid.UUID
	version int
	changes []*domain.Event
}

// New creates an Token
func New() Token {
	return Token{}
}

// FromHistory loads current aggregate root state by applying all events in order
func FromHistory(ctx context.Context, events []*domain.Event) (Token, error) {
	t := New()

	for _, domainEvent := range events {
		var e domain.RawEvent

		switch domainEvent.Type {
		case WasCreatedType:
			e = domainEvent.Payload.(*WasCreated)
		case WasRemovedType:
			e = domainEvent.Payload.(*WasRemoved)
		default:
			return t, apperrors.Wrap(fmt.Errorf("unhandled token event %s", domainEvent.Type))
		}

		if err := t.transition(e); err != nil {
			return t, apperrors.Wrap(err)
		}

		t.version++
	}

	return t, nil
}

// ID returns aggregate root id
func (t Token) ID() uuid.UUID {
	return t.id
}

// Version returns current aggregate root version
func (t Token) Version() int {
	return t.version
}

// Changes returns all new applied events
func (t Token) Changes() []*domain.Event {
	return t.changes
}

// Create alters current token state and append changes to aggregate root
func (t *Token) Create(
	ctx context.Context,
	id uuid.UUID,
	clientID uuid.UUID,
	userID uuid.UUID,
	info oauth2.TokenInfo,
	userAgent string,

) error {
	data, err := json.Marshal(info)
	if err != nil {
		return apperrors.Wrap(err)
	}

	e := &WasCreated{
		ID:        id,
		ClientID:  clientID,
		UserID:    userID,
		Data:      data,
		UserAgent: userAgent,
	}

	if err := t.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	domainEvent, err := domain.NewEventFromRawEvent(t.id, StreamName, t.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if _, err := t.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

// Remove alters current token state and append changes to aggregate root
func (t *Token) Remove(ctx context.Context) error {
	e := &WasRemoved{
		ID: t.id,
	}

	if err := t.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	domainEvent, err := domain.NewEventFromRawEvent(t.id, StreamName, t.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if _, err := t.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

func (t *Token) trackChange(ctx context.Context, event *domain.Event) (*domain.Event, error) {
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

	t.changes = append(t.changes, event)
	t.version++

	return event, nil
}

func (t *Token) transition(e domain.RawEvent) error {
	switch e := e.(type) {
	case *WasCreated:
		t.id = e.ID
		t.userID = e.UserID
	}

	return nil
}
