/*
Package token holds token domain logic
*/
package token

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"gopkg.in/oauth2.v4"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// StreamName for token domain
var StreamName = fmt.Sprintf("%T", Token{})

// Token aggregate root
type Token struct {
	id      uuid.UUID
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

		switch domainEvent.Metadata.Type {
		case (WasCreated{}).GetType():
			wasCreated := WasCreated{}
			if err := unmarshalPayload(domainEvent.Payload, &wasCreated); err != nil {
				log.Panicf("Error while trying to unmarshal token event %s. %s\n", domainEvent.Metadata.Type, err)
			}

			e = wasCreated
		case (WasRemoved{}).GetType():
			wasRemoved := WasRemoved{}
			if err := unmarshalPayload(domainEvent.Payload, &wasRemoved); err != nil {
				log.Panicf("Error while trying to unmarshal token event %s. %s\n", domainEvent.Metadata.Type, err)
			}

			e = wasRemoved
		default:
			log.Panicf("Unhandled token event %s\n", domainEvent.Metadata.Type)
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
func (t *Token) Create(ctx context.Context, id uuid.UUID, info oauth2.TokenInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return errors.Wrap(err)
	}

	clientID, err := uuid.Parse(info.GetClientID())
	if err != nil {
		return errors.Wrap(err)
	}

	userID, err := uuid.Parse(info.GetUserID())
	if err != nil {
		return errors.Wrap(err)
	}

	if _, err = t.trackChange(ctx, WasCreated{
		ID:       id,
		ClientID: clientID,
		UserID:   userID,
		Code:     info.GetCode(),
		Access:   info.GetAccess(),
		Refresh:  info.GetRefresh(),
		Scope:    info.GetScope(),
		Data:     data,
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

	var (
		event domain.Event
		err   error
	)
	if i, hasIdentity := identity.FromContext(ctx); hasIdentity {
		event, err = domain.NewEvent(t.id, StreamName, t.version, e, &i)
	} else {
		event, err = domain.NewEvent(t.id, StreamName, t.version, e, nil)
	}
	if err != nil {
		return event, errors.Wrap(err)
	}

	t.changes = append(t.changes, event)

	return event, nil
}

func (t *Token) transition(e domain.RawEvent) {
	switch e := e.(type) {
	case WasCreated:
		t.id = e.ID
	}
}
