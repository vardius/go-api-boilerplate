/*
Package client holds client domain logic
*/
package client

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
			if err := unmarshalPayload(domainEvent.Payload, &wasCreated); err != nil {
				log.Panicf("Error while trying to unmarshal client event %s. %s\n", domainEvent.Metadata.Type, err)
			}

			e = wasCreated
		case (WasRemoved{}).GetType():
			wasRemoved := WasRemoved{}
			if err := unmarshalPayload(domainEvent.Payload, &wasRemoved); err != nil {
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
func (c *Client) Create(ctx context.Context, info oauth2.ClientInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return errors.Wrap(err)
	}

	id, err := uuid.Parse(info.GetID())
	if err != nil {
		return errors.Wrap(err)
	}

	userID, err := uuid.Parse(info.GetUserID())
	if err != nil {
		return errors.Wrap(err)
	}

	if _, err := c.trackChange(ctx, WasCreated{
		ID:     id,
		UserID: userID,
		Secret: info.GetSecret(),
		Domain: info.GetDomain(),
		Data:   data,
	}); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

// Remove alters current client state and append changes to aggregate root
func (c *Client) Remove(ctx context.Context) error {
	if _, err := c.trackChange(ctx, WasRemoved{
		ID: c.id,
	}); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (c *Client) trackChange(ctx context.Context, e domain.RawEvent) (domain.Event, error) {
	c.transition(e)

	var (
		event domain.Event
		err   error
	)
	if i, hasIdentity := identity.FromContext(ctx); hasIdentity {
		event, err = domain.NewEvent(c.id, StreamName, c.version, e, &i)
	} else {
		event, err = domain.NewEvent(c.id, StreamName, c.version, e, nil)
	}
	if err != nil {
		return event, errors.Wrap(err)
	}

	c.changes = append(c.changes, event)

	return event, nil
}

func (c *Client) transition(e domain.RawEvent) {
	switch e := e.(type) {
	case WasCreated:
		c.id = e.ID
	}
}
