package es

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// Projection updates read models from newly stored events, in the
// transaction that appends them. A failed projection rolls the append back,
// so read models never lag their streams, and a unique index in a read model
// can enforce a rule across aggregates, such as one account per email.
type Projection func(ctx context.Context, tx uow.Tx, events []Event) error

// RepositoryConfig configures a Repository.
type RepositoryConfig[A Aggregate] struct {
	// StreamType names the aggregate's kind, stored with every event: "user".
	StreamType string
	// New returns an empty aggregate to replay events into.
	New      func() A
	Store    Store
	Registry *Registry
	// Project, if set, runs on every Save.
	Project Projection
	// Now stamps events. It defaults to time.Now.
	Now func() time.Time
}

// Repository loads and saves one kind of aggregate.
type Repository[A Aggregate] struct {
	cfg RepositoryConfig[A]
}

// NewRepository returns a Repository for cfg.
func NewRepository[A Aggregate](cfg RepositoryConfig[A]) *Repository[A] {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Repository[A]{cfg: cfg}
}

// Load rebuilds the aggregate with the given ID from its stream. It returns
// an apperr.ErrNotFound when the stream doesn't exist.
func (r *Repository[A]) Load(ctx context.Context, id uuid.UUID) (A, error) {
	var zero A
	records, err := r.cfg.Store.Load(ctx, id)
	if err != nil {
		return zero, fmt.Errorf("load %s %s: %w", r.cfg.StreamType, id, err)
	}
	if len(records) == 0 {
		return zero, fmt.Errorf("load %s %s: %w", r.cfg.StreamType, id, apperr.ErrNotFound)
	}

	a := r.cfg.New()
	for _, rec := range records {
		if rec.StreamType != r.cfg.StreamType {
			return zero, fmt.Errorf("load %s %s: stream holds %s events", r.cfg.StreamType, id, rec.StreamType)
		}
		payload, err := r.cfg.Registry.Decode(rec.Type, rec.Payload)
		if err != nil {
			return zero, fmt.Errorf("load %s %s version %d: %w", r.cfg.StreamType, id, rec.Version, err)
		}
		if err := a.Apply(payload); err != nil {
			return zero, fmt.Errorf("load %s %s version %d: %w", r.cfg.StreamType, id, rec.Version, err)
		}
	}
	a.changes().version = records[len(records)-1].Version
	return a, nil
}

// Save appends the events a raised as part of tx, runs the projection on
// them and returns them. It returns ErrConcurrency when the stream changed
// since a was loaded.
//
// Save leaves a as it was, so a backend that retries the transaction saves
// the same events again. Load the aggregate again to change it further.
func (r *Repository[A]) Save(ctx context.Context, tx uow.Tx, a A, meta Meta) ([]Event, error) {
	c := a.changes()
	if len(c.pending) == 0 {
		return nil, nil
	}

	now := r.cfg.Now().UTC().Truncate(time.Millisecond)
	events := make([]Event, len(c.pending))
	records := make([]Record, len(c.pending))
	for i, payload := range c.pending {
		name, data, err := r.cfg.Registry.Encode(payload)
		if err != nil {
			return nil, fmt.Errorf("save %s %s: %w", r.cfg.StreamType, a.ID(), err)
		}
		events[i] = Event{
			ID:         uuid.NewV7(),
			StreamID:   a.ID(),
			StreamType: r.cfg.StreamType,
			Version:    c.version + i + 1,
			Type:       name,
			OccurredAt: now,
			Payload:    payload,
			Meta:       meta,
		}
		e := events[i]
		records[i] = Record{
			ID:         e.ID,
			StreamID:   e.StreamID,
			StreamType: e.StreamType,
			Version:    e.Version,
			Type:       e.Type,
			OccurredAt: e.OccurredAt,
			Payload:    data,
			Meta:       e.Meta,
		}
	}

	if err := r.cfg.Store.Append(ctx, tx, a.ID(), c.version, records); err != nil {
		return nil, fmt.Errorf("save %s %s: %w", r.cfg.StreamType, a.ID(), err)
	}
	if r.cfg.Project != nil {
		if err := r.cfg.Project(ctx, tx, events); err != nil {
			return nil, fmt.Errorf("project %s %s: %w", r.cfg.StreamType, a.ID(), err)
		}
	}
	return events, nil
}
