/*
Package token holds token domain logic
*/
package token

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	oauth2 "gopkg.in/oauth2.v3"
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
			err := unmarshalPayload(domainEvent.Payload, &wasCreated)
			if err != nil {
				log.Panicf("Error while trying to unmarshal token event %s. %s", domainEvent.Metadata.Type, err)
			}

			e = wasCreated
		case (WasRemoved{}).GetType():
			wasRemoved := WasRemoved{}
			err := unmarshalPayload(domainEvent.Payload, &wasRemoved)
			if err != nil {
				log.Panicf("Error while trying to unmarshal token event %s. %s", domainEvent.Metadata.Type, err)
			}

			e = wasRemoved
		default:
			log.Panicf("Unhandled token event %s", domainEvent.Metadata.Type)
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
func (t *Token) Create(id uuid.UUID, info oauth2.TokenInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Token create error when parsing info to JSON")
	}

	return t.trackChange(WasCreated{
		ID:       id,
		ClientID: uuid.MustParse(info.GetClientID()),
		UserID:   uuid.MustParse(info.GetUserID()),
		Code:     info.GetCode(),
		Access:   info.GetAccess(),
		Refresh:  info.GetRefresh(),
		Scope:    info.GetScope(),
		Data:     data,
	})
}

// Remove alters current token state and append changes to aggregate root
func (t *Token) Remove() error {
	return t.trackChange(WasRemoved{
		ID: t.id,
	})
}

func (t *Token) trackChange(e domain.RawEvent) error {
	t.transition(e)

	event, err := domain.NewEvent(t.id, StreamName, t.version, e)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Token trackChange error")
	}

	t.changes = append(t.changes, event)

	return nil
}

func (t *Token) transition(e domain.RawEvent) {
	switch e := e.(type) {
	case WasCreated:
		t.id = e.ID
	}
}
