// Package outboxtest holds the contract every outbox.Store must meet.
package outboxtest

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/outbox"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// OpenStore returns an empty store and the unit of work its transactions
// come from. Each call must return a store that shares no data with others.
type OpenStore func(t *testing.T) (outbox.Store, uow.UnitOfWork)

// TestStore checks the rules every outbox.Store must follow.
func TestStore(t *testing.T, open OpenStore) {
	t.Run("ClaimReturnsAddedMessage", func(t *testing.T) { testClaim(t, open) })
	t.Run("RollbackDiscardsMessage", func(t *testing.T) { testRollback(t, open) })
	t.Run("LeaseHidesMessageUntilItEnds", func(t *testing.T) { testLease(t, open) })
	t.Run("AckRemovesMessage", func(t *testing.T) { testAck(t, open) })
	t.Run("RetryDelaysMessage", func(t *testing.T) { testRetry(t, open) })
	t.Run("BuryStopsDelivery", func(t *testing.T) { testBury(t, open) })
	t.Run("ClaimTakesEarliestDueFirstUpToLimit", func(t *testing.T) { testOrderAndLimit(t, open) })
	t.Run("MessageIsDueAtItsCreation", func(t *testing.T) { testDueAtCreation(t, open) })
	t.Run("ConcurrentClaimsNeverShare", func(t *testing.T) { testConcurrentClaims(t, open) })
}

// now is a fixed clock with the millisecond precision stores keep.
var now = time.Date(2026, 10, 24, 9, 30, 15, 123_000_000, time.UTC)

const lease = time.Minute

func message(topic string, n int, created time.Time) outbox.Message {
	m, err := outbox.NewMessage(topic, map[string]any{"n": n, "text": "zażółć ✓"}, created)
	if err != nil {
		panic(err)
	}
	return m
}

func add(t *testing.T, store outbox.Store, u uow.UnitOfWork, msgs ...outbox.Message) {
	t.Helper()
	err := u.Do(t.Context(), func(tx uow.Tx) error { return store.Add(t.Context(), tx, msgs...) })
	if err != nil {
		t.Fatalf("add: %v", err)
	}
}

func claim(t *testing.T, store outbox.Store, at time.Time, limit int) []outbox.Message {
	t.Helper()
	msgs, err := store.Claim(t.Context(), at, lease, limit)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	return msgs
}

func ids(msgs []outbox.Message) []uuid.UUID {
	out := make([]uuid.UUID, len(msgs))
	for i, m := range msgs {
		out[i] = m.ID
	}
	return out
}

func assertClaimed(t *testing.T, got []outbox.Message, want ...outbox.Message) {
	t.Helper()
	if !slices.Equal(ids(got), ids(want)) {
		t.Fatalf("claimed %v, want %v", ids(got), ids(want))
	}
}

func testClaim(t *testing.T, open OpenStore) {
	store, u := open(t)
	m := message("mail.login_link", 1, now)
	add(t, store, u, m)

	got := claim(t, store, now, 10)
	assertClaimed(t, got, m)
	c := got[0]
	if c.Topic != m.Topic || string(c.Payload) != string(m.Payload) || !c.CreatedAt.Equal(m.CreatedAt) {
		t.Errorf("claimed %+v, want %+v", c, m)
	}
	if c.Attempts != 1 {
		t.Errorf("attempts = %d, want 1", c.Attempts)
	}
}

func testRollback(t *testing.T, open OpenStore) {
	store, u := open(t)
	failure := errors.New("the change failed")
	err := u.Do(t.Context(), func(tx uow.Tx) error {
		if err := store.Add(t.Context(), tx, message("mail.login_link", 1, now)); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("Do returned %v, want the function's error", err)
	}
	assertClaimed(t, claim(t, store, now, 10))
}

func testLease(t *testing.T, open OpenStore) {
	store, u := open(t)
	m := message("mail.login_link", 1, now)
	add(t, store, u, m)
	claim(t, store, now, 10)

	assertClaimed(t, claim(t, store, now.Add(lease-time.Millisecond), 10))

	// The relay that claimed it never acked: it crashed. The message comes back.
	got := claim(t, store, now.Add(lease), 10)
	assertClaimed(t, got, m)
	if got[0].Attempts != 2 {
		t.Errorf("attempts = %d, want 2", got[0].Attempts)
	}
}

func testAck(t *testing.T, open OpenStore) {
	store, u := open(t)
	m := message("mail.login_link", 1, now)
	add(t, store, u, m)
	claim(t, store, now, 10)

	if err := store.Ack(t.Context(), m.ID); err != nil {
		t.Fatalf("ack: %v", err)
	}
	assertClaimed(t, claim(t, store, now.Add(2*lease), 10))
}

func testRetry(t *testing.T, open OpenStore) {
	store, u := open(t)
	m := message("mail.login_link", 1, now)
	add(t, store, u, m)
	claim(t, store, now, 10)

	retryAt := now.Add(10 * time.Second)
	if err := store.Retry(t.Context(), m.ID, retryAt, "smtp: connection refused"); err != nil {
		t.Fatalf("retry: %v", err)
	}
	assertClaimed(t, claim(t, store, retryAt.Add(-time.Millisecond), 10))

	// The retry ended the lease, so the message is due at retryAt, not at the lease's end.
	got := claim(t, store, retryAt, 10)
	assertClaimed(t, got, m)
	if got[0].Attempts != 2 {
		t.Errorf("attempts = %d, want 2", got[0].Attempts)
	}
}

func testBury(t *testing.T, open OpenStore) {
	store, u := open(t)
	buried, other := message("mail.login_link", 1, now), message("mail.login_link", 2, now.Add(time.Millisecond))
	add(t, store, u, buried, other)
	claim(t, store, now, 1)

	if err := store.Bury(t.Context(), buried.ID, "template missing"); err != nil {
		t.Fatalf("bury: %v", err)
	}
	assertClaimed(t, claim(t, store, now.Add(24*time.Hour), 10), other)
}

func testOrderAndLimit(t *testing.T, open OpenStore) {
	store, u := open(t)
	var msgs []outbox.Message
	for i := range 5 {
		msgs = append(msgs, message("mail.login_link", i, now.Add(time.Duration(i)*time.Millisecond)))
	}
	add(t, store, u, msgs[3], msgs[0], msgs[4])
	add(t, store, u, msgs[1], msgs[2])

	at := now.Add(time.Second)
	assertClaimed(t, claim(t, store, at, 2), msgs[0], msgs[1])
	assertClaimed(t, claim(t, store, at, 2), msgs[2], msgs[3])
	assertClaimed(t, claim(t, store, at, 2), msgs[4])
}

func testDueAtCreation(t *testing.T, open OpenStore) {
	store, u := open(t)
	m := message("mail.login_link", 1, now.Add(time.Minute))
	add(t, store, u, m)

	assertClaimed(t, claim(t, store, now, 10))
	assertClaimed(t, claim(t, store, now.Add(time.Minute), 10), m)
}

func testConcurrentClaims(t *testing.T, open OpenStore) {
	store, u := open(t)
	const total = 40
	var msgs []outbox.Message
	for i := range total {
		msgs = append(msgs, message("mail.login_link", i, now.Add(time.Duration(i)*time.Millisecond)))
	}
	add(t, store, u, msgs...)

	const claimers = 8
	var (
		mu   sync.Mutex
		seen = map[uuid.UUID]int{}
		wg   sync.WaitGroup
		errs = make(chan error, claimers)
	)
	at := now.Add(time.Second)
	for range claimers {
		wg.Go(func() {
			for {
				got, err := store.Claim(context.WithoutCancel(t.Context()), at, lease, 3)
				if err != nil {
					errs <- err
					return
				}
				if len(got) == 0 {
					return
				}
				mu.Lock()
				for _, m := range got {
					seen[m.ID]++
				}
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent claim: %v", err)
	}

	if len(seen) != total {
		t.Errorf("claimed %d distinct messages, want %d", len(seen), total)
	}
	for id, n := range seen {
		if n > 1 {
			t.Errorf("message %s was claimed %d times", id, n)
		}
	}
	if t.Failed() {
		t.Log(fmt.Sprint("claims by message: ", seen))
	}
}
