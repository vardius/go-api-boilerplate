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

	authdomain "github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
	"github.com/vardius/go-api-boilerplate/pkg/metadata"
)

// StreamName for client domain
var StreamName = fmt.Sprintf("%T", Client{})

// Client aggregate root
type Client struct {
	id      uuid.UUID
	userID  uuid.UUID
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

		switch domainEvent.Type {
		case (WasCreated{}).GetType():
			wasCreated := WasCreated{}
			if err := json.Unmarshal(domainEvent.Payload, &wasCreated); err != nil {
				log.Panicf("Error while trying to unmarshal client event %s. %s", domainEvent.Type, err)
			}

			e = wasCreated
		case (WasRemoved{}).GetType():
			wasRemoved := WasRemoved{}
			if err := json.Unmarshal(domainEvent.Payload, &wasRemoved); err != nil {
				log.Panicf("Error while trying to unmarshal client event %s. %s", domainEvent.Type, err)
			}

			e = wasRemoved
		default:
			log.Panicf("Unhandled client event %s", domainEvent.Type)
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
func (c *Client) Create(
	ctx context.Context,
	clientID uuid.UUID,
	clientSecret uuid.UUID,
	userID uuid.UUID,
	domain string,
	redirectURL string,
	scopes ...string,
) error {
	if _, err := c.trackChange(ctx, WasCreated{
		ID:          clientID,
		Secret:      clientSecret,
		UserID:      userID,
		Domain:      domain,
		RedirectURL: redirectURL,
		Scopes:      scopes,
	}); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

// Remove alters current client state and append changes to aggregate root
func (c *Client) Remove(ctx context.Context) error {
	if _, err := c.trackChange(ctx, WasRemoved{
		ID: c.id,
	}); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

func (c *Client) trackChange(ctx context.Context, e domain.RawEvent) (domain.Event, error) {
	c.transition(e)

	event, err := domain.NewEventFromRawEvent(c.id, StreamName, c.version, e)
	if err != nil {
		return event, apperrors.Wrap(err)
	}

	meta := authdomain.EventMetadata{}
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

	c.changes = append(c.changes, event)

	return event, nil
}

func (c *Client) transition(e domain.RawEvent) {
	switch e := e.(type) {
	case WasCreated:
		c.id = e.ID
		c.userID = e.UserID
	}
}
