/*
Package client holds client domain logic
*/
package client

import (
	"context"
	"fmt"

	"github.com/google/uuid"

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
	changes []*domain.Event
}

// New creates an Client
func New() Client {
	return Client{}
}

// FromHistory loads current aggregate root state by applying all events in order
func FromHistory(ctx context.Context, events []*domain.Event) (Client, error) {
	c := New()

	for _, domainEvent := range events {
		var e domain.RawEvent

		switch domainEvent.Type {
		case WasCreatedType:
			e = domainEvent.Payload.(*WasCreated)
		case WasRemovedType:
			e = domainEvent.Payload.(*WasRemoved)
		default:
			return c, apperrors.Wrap(fmt.Errorf("unhandled client event %s", domainEvent.Type))
		}

		if err := c.transition(e); err != nil {
			return c, apperrors.Wrap(err)
		}

		c.version++
	}

	return c, nil
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
func (c Client) Changes() []*domain.Event {
	return c.changes
}

// Create alters current client state and append changes to aggregate root
func (c *Client) Create(
	ctx context.Context,
	clientID uuid.UUID,
	clientSecret uuid.UUID,
	userID uuid.UUID,
	domainName string,
	redirectURL string,
	scopes ...string,
) error {
	e := &WasCreated{
		ID:          clientID,
		Secret:      clientSecret,
		UserID:      userID,
		Domain:      domainName,
		RedirectURL: redirectURL,
		Scopes:      scopes,
	}

	if err := c.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	domainEvent, err := domain.NewEventFromRawEvent(c.id, StreamName, c.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if _, err := c.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

// Remove alters current client state and append changes to aggregate root
func (c *Client) Remove(ctx context.Context) error {
	e := &WasRemoved{
		ID: c.id,
	}

	if err := c.transition(e); err != nil {
		return apperrors.Wrap(err)
	}

	domainEvent, err := domain.NewEventFromRawEvent(c.id, StreamName, c.version, e)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if _, err := c.trackChange(ctx, domainEvent); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

func (c *Client) trackChange(ctx context.Context, event *domain.Event) (*domain.Event, error) {
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

	c.changes = append(c.changes, event)
	c.version++

	return event, nil
}

func (c *Client) transition(e domain.RawEvent) error {
	switch e := e.(type) {
	case *WasCreated:
		c.id = e.ID
		c.userID = e.UserID
	}

	return nil
}
