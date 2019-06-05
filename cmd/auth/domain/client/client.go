/*
Package client holds client domain logic
*/
package client

import (
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	oauth2 "gopkg.in/oauth2.v3"
)

// StreamName for client domain
const StreamName = "client" //fmt.Sprintf("%T", Client{})

// Client aggregate root
type Client struct {
	id      uuid.UUID
	version int
	changes []*domain.Event
}

func (c *Client) transition(e domain.RawEvent) {
	switch e := e.(type) {
	case *WasCreated:
		c.id = e.ID
	}
}

func (c *Client) trackChange(e domain.RawEvent) error {
	c.transition(e)
	eventEnvelop, err := domain.NewEvent(c.id, StreamName, c.version, e)

	if err != nil {
		return err
	}

	c.changes = append(c.changes, eventEnvelop)

	return nil
}

// ID returns aggregate root id
func (c *Client) ID() uuid.UUID {
	return c.id
}

// Version returns current aggregate root version
func (c *Client) Version() int {
	return c.version
}

// Changes returns all new applied events
func (c *Client) Changes() []*domain.Event {
	return c.changes
}

// Create alters current user state and append changes to aggregate root
func (c *Client) Create(info oauth2.ClientInfo) error {
	return c.trackChange(&WasCreated{
		ID:     uuid.MustParse(info.GetID()),
		UserID: uuid.MustParse(info.GetUserID()),
		Info:   info,
	})
}

// Remove alters current user state and append changes to aggregate root
func (c *Client) Remove() error {
	return c.trackChange(&WasRemoved{
		ID: c.id,
	})
}

// FromHistory loads current aggregate root state by applying all events in order
func (c *Client) FromHistory(events []*domain.Event) {
	for _, domainEvent := range events {
		var e domain.RawEvent

		switch domainEvent.Metadata.Type {
		case (&WasCreated{}).GetType():
			e = &WasCreated{}
		case (&WasRemoved{}).GetType():
			e = &WasRemoved{}
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

		c.transition(e)
		c.version++
	}
}

// New creates an Client
func New() *Client {
	return &Client{}
}
