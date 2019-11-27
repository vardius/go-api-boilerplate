/*
Package client holds client domain logic
*/
package client

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/internal/domain"
	"github.com/vardius/go-api-boilerplate/internal/errors"
	oauth2 "gopkg.in/oauth2.v3"
)

// StreamName for client domain
var StreamName = fmt.Sprintf("%T", Client{})

// Client aggregate root
type Client struct {
	id      uuid.UUID
	version int
	changes []domain.Event
}

// New creates an Client
func New() Client {
	return Client{}
}

// FromHistory loads current aggregate root state by applying all events in order
func FromHistory(events []domain.Event) Client {
	c := New()

	for _, domainEvent := range events {
		var e domain.RawEvent

		switch domainEvent.Metadata.Type {
		case (WasCreated{}).GetType():
			wasCreated := WasCreated{}
			err := unmarshalPayload(domainEvent.Payload, &wasCreated)
			if err != nil {
				log.Panicf("Error while trying to unmarshal client event %s. %s\n", domainEvent.Metadata.Type, err)
			}

			e = wasCreated
		case (WasRemoved{}).GetType():
			wasRemoved := WasRemoved{}
			err := unmarshalPayload(domainEvent.Payload, &wasRemoved)
			if err != nil {
				log.Panicf("Error while trying to unmarshal client event %s. %s\n", domainEvent.Metadata.Type, err)
			}

			e = wasRemoved
		default:
			log.Panicf("Unhandled client event %s\n", domainEvent.Metadata.Type)
		}

		c.transition(e)
		c.version++
	}

	return c
}

// ID returns aggregate root id
func (c Client) ID() uuid.UUID {
	return c.id
}

// Version returns current aggregate root version
func (c Client) Version() int {
	return c.version
}

// Changes returns all new applied events
func (c Client) Changes() []domain.Event {
	return c.changes
}

// Create alters current client state and append changes to aggregate root
func (c *Client) Create(info oauth2.ClientInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Client create error when parsing info to JSON")
	}

	return c.trackChange(WasCreated{
		ID:     uuid.MustParse(info.GetID()),
		UserID: uuid.MustParse(info.GetUserID()),
		Secret: info.GetSecret(),
		Domain: info.GetDomain(),
		Data:   data,
	})
}

// Remove alters current client state and append changes to aggregate root
func (c *Client) Remove() error {
	return c.trackChange(WasRemoved{
		ID: c.id,
	})
}

func (c *Client) trackChange(e domain.RawEvent) error {
	c.transition(e)

	event, err := domain.NewEvent(c.id, StreamName, c.version, e)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Client trackChange error")
	}

	c.changes = append(c.changes, event)

	return nil
}

func (c *Client) transition(e domain.RawEvent) {
	switch e := e.(type) {
	case WasCreated:
		c.id = e.ID
	}
}
