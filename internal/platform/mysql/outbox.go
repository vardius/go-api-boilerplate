package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/outbox"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// Outbox is an outbox.Store in the outbox table.
type Outbox struct {
	db *DB
}

var _ outbox.Store = (*Outbox)(nil)

// NewOutbox returns an outbox on db.
func NewOutbox(db *DB) *Outbox {
	return &Outbox{db: db}
}

// Add implements outbox.Store. A message whose ID is already stored is an
// apperr.ErrConflict.
func (o *Outbox) Add(ctx context.Context, tx uow.Tx, msgs ...outbox.Message) error {
	t, err := SQLTx(tx)
	if err != nil {
		return err
	}
	if len(msgs) == 0 {
		return nil
	}

	args := make([]any, 0, 5*len(msgs))
	for _, m := range msgs {
		args = append(args, m.ID[:], m.Topic, string(m.Payload), m.CreatedAt, m.CreatedAt)
	}
	query := "INSERT INTO outbox (id, topic, payload, created_at, due_at) VALUES " + valueGroups(len(msgs), "(?, ?, ?, ?, ?)")
	if _, err := t.ExecContext(ctx, query, args...); err != nil {
		if IsDuplicate(err) {
			return fmt.Errorf("mysql: add outbox messages: %w: %w", apperr.ErrConflict, err)
		}
		return fmt.Errorf("mysql: add outbox messages: %w", err)
	}
	return nil
}

// Claim implements outbox.Store. It locks due rows with FOR UPDATE SKIP
// LOCKED, so concurrent claims pass over each other's rows instead of
// waiting for them.
func (o *Outbox) Claim(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]outbox.Message, error) {
	if limit <= 0 {
		return nil, nil
	}
	// Under READ COMMITTED InnoDB takes no gap locks, so a claim never holds
	// up a transaction adding a message, and it lets go of the rows it reads
	// but doesn't claim.
	opts := &sql.TxOptions{Isolation: sql.LevelReadCommitted}
	var claimed []outbox.Message
	err := o.db.do(ctx, opts, func(tx *sql.Tx) error {
		msgs, err := lockDue(ctx, tx, now, limit)
		if err != nil {
			return fmt.Errorf("mysql: claim outbox messages: %w", err)
		}
		if len(msgs) == 0 {
			return nil
		}

		args := make([]any, 0, 1+len(msgs))
		args = append(args, now.Add(lease))
		for _, m := range msgs {
			args = append(args, m.ID[:])
		}
		query := "UPDATE outbox SET attempts = attempts + 1, claimed_until = ? WHERE id IN (" + valueGroups(len(msgs), "?") + ")"
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("mysql: lease outbox messages: %w", err)
		}
		for i := range msgs {
			msgs[i].Attempts++
		}
		claimed = msgs
		return nil
	})
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

// lockDue locks and returns up to limit messages due at now, earliest due
// first. Messages due at the same time come in ID order, which for UUIDv7
// IDs is the order they were made in.
func lockDue(ctx context.Context, tx *sql.Tx, now time.Time, limit int) ([]outbox.Message, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, topic, payload, created_at, attempts FROM outbox
		WHERE buried_at IS NULL AND due_at <= ? AND (claimed_until IS NULL OR claimed_until <= ?)
		ORDER BY due_at, id
		LIMIT ?
		FOR UPDATE SKIP LOCKED`,
		now, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []outbox.Message
	for rows.Next() {
		var m outbox.Message
		if err := rows.Scan(uuidColumn{&m.ID}, &m.Topic, &m.Payload, &m.CreatedAt, &m.Attempts); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return msgs, nil
}

// Ack implements outbox.Store. Acking a message that is already gone does
// nothing.
func (o *Outbox) Ack(ctx context.Context, id uuid.UUID) error {
	if _, err := o.db.ExecContext(ctx, "DELETE FROM outbox WHERE id = ?", id[:]); err != nil {
		return fmt.Errorf("mysql: ack outbox message %s: %w", id, err)
	}
	return nil
}

// Retry implements outbox.Store. An unknown message is an
// apperr.ErrNotFound.
func (o *Outbox) Retry(ctx context.Context, id uuid.UUID, at time.Time, reason string) error {
	return o.update(ctx, "retry", id,
		"UPDATE outbox SET due_at = ?, claimed_until = NULL, last_error = ? WHERE id = ?",
		at, storedReason(reason), id[:])
}

// Bury implements outbox.Store. The row stays, with buried_at set to the
// server's clock and reason as last_error. An unknown message is an
// apperr.ErrNotFound.
func (o *Outbox) Bury(ctx context.Context, id uuid.UUID, reason string) error {
	return o.update(ctx, "bury", id,
		"UPDATE outbox SET buried_at = UTC_TIMESTAMP(3), last_error = ? WHERE id = ?",
		storedReason(reason), id[:])
}

func (o *Outbox) update(ctx context.Context, verb string, id uuid.UUID, query string, args ...any) error {
	res, err := o.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("mysql: %s outbox message %s: %w", verb, id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mysql: %s outbox message %s: %w", verb, id, err)
	}
	if n == 0 {
		return fmt.Errorf("mysql: %s outbox message %s: %w", verb, id, apperr.ErrNotFound)
	}
	return nil
}

// maxReasonBytes bounds a stored last_error well inside its TEXT column.
const maxReasonBytes = 4 << 10

// storedReason makes reason fit last_error. A handler's error can carry a
// whole response body or raw bytes, and a value the column rejects would
// make the message impossible to retry or bury.
func storedReason(reason string) string {
	reason = strings.ToValidUTF8(reason, "�")
	if len(reason) <= maxReasonBytes {
		return reason
	}
	cut := maxReasonBytes
	for !utf8.RuneStart(reason[cut]) {
		cut--
	}
	return reason[:cut]
}
