/*
Package token holds token domain logic
*/
package token

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"

	authdomain "github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
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
	changes []domain.Event
}

// New creates an Token
func New() Token {
	return Token{}
}

// FromHistory loads current aggregate root state by applying all events in order
func FromHistory(events []domain.Event) Token {
	t := New()

	for _, domainEvent := range events {
		var e domain.RawEvent

		switch domainEvent.Type {
		case (WasCreated{}).GetType():
			wasCreated := WasCreated{}
			if err := unmarshalPayload(domainEvent.Payload, &wasCreated); err != nil {
				log.Panicf("Error while trying to unmarshal token event %s. %s", domainEvent.Type, err)
			}

			e = wasCreated
		case (WasRemoved{}).GetType():
			wasRemoved := WasRemoved{}
			if err := unmarshalPayload(domainEvent.Payload, &wasRemoved); err != nil {
				log.Panicf("Error while trying to unmarshal token event %s. %s", domainEvent.Type, err)
			}

			e = wasRemoved
		default:
			log.Panicf("Unhandled token event %s", domainEvent.Type)
		}

		t.transition(e)
		t.version++
	}

	return t
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
func (t Token) Changes() []domain.Event {
	return t.changes
}

// Create alters current token state and append changes to aggregate root
func (t *Token) Create(
	ctx context.Context,
	id uuid.UUID,
	clientID uuid.UUID,
	userID uuid.UUID,
	code string,
	scope string,
	access string,
	refresh string,

) error {
	if _, err := t.trackChange(ctx, WasCreated{
		ID:       id,
		ClientID: clientID,
		UserID:   userID,
		Code:     code,
		Access:   access,
		Refresh:  refresh,
		Scope:    scope,
	}); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

// Remove alters current token state and append changes to aggregate root
func (t *Token) Remove(ctx context.Context) error {
	if _, err := t.trackChange(ctx, WasRemoved{
		ID: t.id,
	}); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (t *Token) trackChange(ctx context.Context, e domain.RawEvent) (domain.Event, error) {
	t.transition(e)

	event, err := domain.NewEventFromRawEvent(t.id, StreamName, t.version, e)
	if err != nil {
		return event, errors.Wrap(err)
	}

	meta := authdomain.EventMetadata{}
	if i, hasIdentity := identity.FromContext(ctx); hasIdentity {
		meta.Identity = &i
	}
	if m, ok := metadata.FromContext(ctx); ok {
		meta.IPAddress = m.IPAddress
	}
	if !meta.IsEmpty() {
		if err := event.WithMetadata(meta); err != nil {
			return event, errors.Wrap(err)
		}
	}

	t.changes = append(t.changes, event)

	return event, nil
}

func (t *Token) transition(e domain.RawEvent) {
	switch e := e.(type) {
	case WasCreated:
		t.id = e.ID
		t.userID = e.UserID
	}
}
