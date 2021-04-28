package eventstore

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
	"go.mongodb.org/mongo-driver/bson"
	"net"
	"time"
)

type DTO struct {
	ID            string            `bson:"event_id"`
	Type          string            `bson:"event_type"`
	StreamID      string            `bson:"stream_id"`
	StreamName    string            `bson:"stream_name"`
	StreamVersion int               `bson:"stream_version"`
	OccurredAt    time.Time         `bson:"occurred_at"`
	ExpiresAt     *time.Time        `bson:"expires_at,omitempty"`
	Payload       bson.Raw          `bson:"payload"`
	Metadata      *EventMetadataDTO `bson:"metadata,omitempty"`
}

type EventMetadataDTO struct {
	Identity  *identity.Identity `bson:"identity,omitempty"`
	IPAddress net.IP             `bson:"ip_address,omitempty"`
	UserAgent string             `bson:"http_user_agent,omitempty"`
	Referer   string             `bson:"http_referer,omitempty"`
}

func (o *DTO) ToEvent() (*domain.Event, error) {
	rawEvent, err := domain.NewRawEvent(o.Type)
	if err != nil {
		return nil, apperrors.Wrap(fmt.Errorf("failed to create raw event:%s: %w", o.Type, err))
	}
	if err := bson.Unmarshal(o.Payload, rawEvent); err != nil {
		return nil, apperrors.Wrap(fmt.Errorf("failed to unmarshal raw event:%s: %w", o.Type, err))
	}

	id, err := uuid.Parse(o.ID)
	if err != nil {
		return nil, apperrors.Wrap(fmt.Errorf("failed to parse id:%s: %w", o.ID, err))
	}
	streamID, err := uuid.Parse(o.StreamID)
	if err != nil {
		return nil, apperrors.Wrap(fmt.Errorf("failed to parse strem id:%s: %w", o.StreamID, err))
	}

	event := &domain.Event{
		ID:            id,
		StreamID:      streamID,
		Type:          o.Type,
		StreamName:    o.StreamName,
		StreamVersion: o.StreamVersion,
		OccurredAt:    o.OccurredAt,
		ExpiresAt:     o.ExpiresAt,
		Payload:       rawEvent,
	}

	if o.Metadata != nil {
		event.Metadata = &domain.EventMetadata{
			Identity:  o.Metadata.Identity,
			IPAddress: o.Metadata.IPAddress,
			UserAgent: o.Metadata.UserAgent,
			Referer:   o.Metadata.Referer,
		}
	}

	return event, nil
}

func NewDTOFromEvent(e *domain.Event) (*DTO, error) {
	payload, err := bson.Marshal(e.Payload)
	if err != nil {
		return nil, apperrors.Wrap(fmt.Errorf("failed to marshal raw event:%s: %w", e.Type, err))
	}
	dto := &DTO{
		ID:            e.ID.String(),
		Type:          e.Type,
		StreamID:      e.StreamID.String(),
		StreamName:    e.StreamName,
		StreamVersion: e.StreamVersion,
		OccurredAt:    e.OccurredAt,
		ExpiresAt:     e.ExpiresAt,
		Payload:       payload,
	}

	if e.Metadata != nil {
		dto.Metadata = &EventMetadataDTO{
			Identity:  e.Metadata.Identity,
			IPAddress: e.Metadata.IPAddress,
			UserAgent: e.Metadata.UserAgent,
			Referer:   e.Metadata.Referer,
		}
	}
	return dto, nil
}
