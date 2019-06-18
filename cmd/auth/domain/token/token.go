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

func (t Token) transition(e domain.RawEvent) {
	switch e := e.(type) {
	case WasCreated:
		t.id = e.ID
	}
}

func (t Token) trackChange(e domain.RawEvent) error {
	t.transition(e)
	event, err := domain.NewEvent(t.id, StreamName, t.version, e)

	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Token trackChange error")
	}

	t.changes = append(t.changes, event)

	return nil
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

// Create alters current user state and append changes to aggregate root
func (t Token) Create(id uuid.UUID, info oauth2.TokenInfo) error {
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

// Remove alters current user state and append changes to aggregate root
func (t Token) Remove() error {
	return t.trackChange(WasRemoved{
		ID: t.id,
	})
}

// FromHistory loads current aggregate root state by applying all events in order
func (t Token) FromHistory(events []domain.Event) {
	for _, domainEvent := range events {
		var e domain.RawEvent

		switch domainEvent.Metadata.Type {
		case (WasCreated{}).GetType():
			e = WasCreated{}
		case (WasRemoved{}).GetType():
			e = WasRemoved{}
		default:
			// @TODO: should we panic here ?
			log.Panicf("Unhandled user event %s", domainEvent.Metadata.Type)
		}

		err := json.Unmarshal(domainEvent.Payload, e)
		if err != nil {
			// @TODO: should we panic here ?
			log.Panicf("Error while parsing json to a user event %s, %s", domainEvent.Metadata.Type, domainEvent.Payload)
			continue
		}

		t.transition(e)
		t.version++
	}
}

// New creates an Token
func New() Token {
	return Token{}
}
