package memdb

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/outbox"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// Outbox is an outbox.Store in memory.
type Outbox struct {
	db   *DB
	msgs map[uuid.UUID]*message
	seq  int
}

type message struct {
	outbox.Message
	seq          int
	dueAt        time.Time
	claimedUntil time.Time
	buried       bool
	lastError    string
}

var _ outbox.Store = (*Outbox)(nil)

// NewOutbox returns an empty outbox on db.
func NewOutbox(db *DB) *Outbox {
	return &Outbox{db: db, msgs: map[uuid.UUID]*message{}}
}

// Add implements outbox.Store.
func (o *Outbox) Add(_ context.Context, tx uow.Tx, msgs ...outbox.Message) error {
	t, err := From(tx)
	if err != nil {
		return err
	}
	for _, m := range msgs {
		if _, ok := o.msgs[m.ID]; ok {
			return fmt.Errorf("memdb: outbox message %s: %w", m.ID, apperr.ErrConflict)
		}
		o.seq++
		m.Payload = slices.Clone(m.Payload)
		m.Attempts = 0
		o.msgs[m.ID] = &message{Message: m, seq: o.seq, dueAt: m.CreatedAt}
		t.OnRollback(func() { delete(o.msgs, m.ID) })
	}
	return nil
}

// Claim implements outbox.Store.
func (o *Outbox) Claim(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]outbox.Message, error) {
	var claimed []outbox.Message
	err := o.db.Do(ctx, func(uow.Tx) error {
		var due []*message
		for _, m := range o.msgs {
			if !m.buried && !m.dueAt.After(now) && !m.claimedUntil.After(now) {
				due = append(due, m)
			}
		}
		slices.SortFunc(due, func(a, b *message) int {
			return cmp.Or(a.dueAt.Compare(b.dueAt), cmp.Compare(a.seq, b.seq))
		})
		for _, m := range due[:min(limit, len(due))] {
			m.Attempts++
			m.claimedUntil = now.Add(lease)
			c := m.Message
			c.Payload = slices.Clone(c.Payload)
			claimed = append(claimed, c)
		}
		return nil
	})
	return claimed, err
}

// Ack implements outbox.Store.
func (o *Outbox) Ack(ctx context.Context, id uuid.UUID) error {
	return o.db.Do(ctx, func(uow.Tx) error {
		delete(o.msgs, id)
		return nil
	})
}

// Retry implements outbox.Store.
func (o *Outbox) Retry(ctx context.Context, id uuid.UUID, at time.Time, reason string) error {
	return o.update(ctx, id, func(m *message) {
		m.dueAt = at
		m.claimedUntil = time.Time{}
		m.lastError = reason
	})
}

// Bury implements outbox.Store.
func (o *Outbox) Bury(ctx context.Context, id uuid.UUID, reason string) error {
	return o.update(ctx, id, func(m *message) {
		m.buried = true
		m.lastError = reason
	})
}

// Buried returns the IDs of buried messages, oldest first, for tests and
// operators.
func (o *Outbox) Buried() []uuid.UUID {
	var ids []uuid.UUID
	_ = o.db.Read(func() error {
		buried := slices.SortedFunc(maps.Values(o.msgs), func(a, b *message) int { return cmp.Compare(a.seq, b.seq) })
		for _, m := range buried {
			if m.buried {
				ids = append(ids, m.ID)
			}
		}
		return nil
	})
	return ids
}

func (o *Outbox) update(ctx context.Context, id uuid.UUID, fn func(*message)) error {
	return o.db.Do(ctx, func(uow.Tx) error {
		m, ok := o.msgs[id]
		if !ok {
			return fmt.Errorf("memdb: outbox message %s: %w", id, apperr.ErrNotFound)
		}
		fn(m)
		return nil
	})
}
