// Package es is a small event-sourcing kit.
//
// An aggregate embeds Changes and changes state only by raising events
// (Raise). A Repository rebuilds it from its stream and appends what it
// raised, checking the stream's version so that two writers can't both
// succeed. Payloads are plain Go values registered under explicit,
// versioned names (Register), never their Go type names, so renaming a
// type can't orphan stored events.
//
// Stores keep encoded Records; the memdb, mysql and mongodb packages
// implement Store, and estest.TestStore holds each of them to the same
// rules.
package es

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// Event is one fact in an aggregate's stream, with its payload decoded.
type Event struct {
	ID         uuid.UUID
	StreamID   uuid.UUID
	StreamType string
	// Version is the event's position in its stream, starting at 1.
	Version    int
	Type       string
	OccurredAt time.Time
	Payload    any
	Meta       Meta
}

// Meta says why an event happened. Events are kept forever, so Meta holds
// identifiers only: never tokens, IP addresses or other personal data.
type Meta struct {
	// CorrelationID ties together everything one request caused.
	CorrelationID string `json:"correlation_id,omitzero" bson:"correlation_id,omitempty"`
	// CausationID is the ID of the event or message that caused this one.
	CausationID string `json:"causation_id,omitzero" bson:"causation_id,omitempty"`
	// ActorID is the user or client that caused the event, if any.
	ActorID string `json:"actor_id,omitzero" bson:"actor_id,omitempty"`
}

// Record is an Event as a Store keeps it, with its payload encoded as JSON.
// OccurredAt has millisecond precision, the finest every backend keeps.
type Record struct {
	ID         uuid.UUID
	StreamID   uuid.UUID
	StreamType string
	Version    int
	Type       string
	OccurredAt time.Time
	Payload    []byte
	Meta       Meta
}

// ErrConcurrency means the stream changed after it was loaded. The caller
// may reload and retry. It is an apperr.ErrConflict.
var ErrConcurrency = apperr.New(apperr.ErrConflict, "this was changed by another request; reload and try again")

// ErrUnknownEventType means a stored event's type isn't registered.
var ErrUnknownEventType = errors.New("unknown event type")

// Store keeps event streams.
type Store interface {
	// Append adds records to the end of a stream as part of tx. Their
	// versions run from expectedVersion+1 up. When the stream's current
	// version isn't expectedVersion, Append writes nothing and returns
	// ErrConcurrency, including when a concurrent transaction appended the
	// same version first.
	Append(ctx context.Context, tx uow.Tx, streamID uuid.UUID, expectedVersion int, records []Record) error
	// Load returns a stream's committed records in version order, or none
	// for a stream that doesn't exist.
	Load(ctx context.Context, streamID uuid.UUID) ([]Record, error)
}
