// Package outbox delivers side effects, such as emails, at least once.
//
// A use case adds a Message in the same transaction as the change that
// causes it, so the message exists exactly when the change does. A Relay
// then claims due messages and hands each to its topic's Handler, retrying
// with backoff and burying a message that keeps failing. A message can be
// delivered more than once (a relay may crash after the handler succeeds),
// so handlers use the message ID to make repeats harmless.
//
// The memdb, mysql and mongodb packages implement Store, and
// outboxtest.TestStore holds each of them to the same rules.
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// Message is one side effect to deliver.
type Message struct {
	ID    uuid.UUID
	Topic string
	// Payload is JSON.
	Payload   []byte
	CreatedAt time.Time
	// Attempts counts the times the message was claimed, this one included.
	// Store.Claim sets it.
	Attempts int
}

// NewMessage returns a message for topic with payload encoded as JSON.
func NewMessage(topic string, payload any, now time.Time) (Message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Message{}, fmt.Errorf("outbox: encode %s: %w", topic, err)
	}
	return Message{
		ID:        uuid.NewV7(),
		Topic:     topic,
		Payload:   data,
		CreatedAt: now.UTC().Truncate(time.Millisecond),
	}, nil
}

// Store keeps messages until they are delivered.
type Store interface {
	// Add stores messages as part of tx. Each becomes due at its CreatedAt.
	Add(ctx context.Context, tx uow.Tx, msgs ...Message) error
	// Claim leases up to limit due messages, earliest due first, until now+lease,
	// and increments their attempts. A due message is one that isn't buried,
	// whose retry time has passed and whose lease, if any, has ended, so a
	// message claimed by a relay that crashed is claimed again. Concurrent
	// claims never return the same message.
	Claim(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]Message, error)
	// Ack removes a delivered message.
	Ack(ctx context.Context, id uuid.UUID) error
	// Retry ends a message's lease and makes it due again at at, keeping
	// reason as its last error.
	Retry(ctx context.Context, id uuid.UUID, at time.Time, reason string) error
	// Bury stops delivering a message and keeps it, with reason, for an
	// operator to inspect.
	Bury(ctx context.Context, id uuid.UUID, reason string) error
}
