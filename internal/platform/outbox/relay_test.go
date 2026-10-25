package outbox_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/memdb"
	"github.com/vardius/go-api-boilerplate/internal/platform/outbox"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

type fixture struct {
	db     *memdb.DB
	store  *memdb.Outbox
	now    time.Time
	relay  *outbox.Relay
	handle func(context.Context, outbox.Message) error
}

func newFixture(t *testing.T) *fixture {
	db := memdb.New()
	f := &fixture{db: db, store: memdb.NewOutbox(db), now: time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)}
	f.relay = &outbox.Relay{
		Store: f.store,
		Handlers: map[string]outbox.Handler{
			"mail": func(ctx context.Context, m outbox.Message) error { return f.handle(ctx, m) },
		},
		Config: outbox.RelayConfig{MaxAttempts: 3},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:    func() time.Time { return f.now },
	}
	return f
}

func (f *fixture) add(t *testing.T, topic string) outbox.Message {
	t.Helper()
	m, err := outbox.NewMessage(topic, map[string]string{"to": "ada@example.com"}, f.now)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.db.Do(t.Context(), func(tx uow.Tx) error { return f.store.Add(t.Context(), tx, m) }); err != nil {
		t.Fatal(err)
	}
	return m
}

func (f *fixture) drain(t *testing.T) {
	t.Helper()
	if _, err := f.relay.Drain(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestRelayRetriesUntilDelivered(t *testing.T) {
	f := newFixture(t)
	var calls []int
	var delivered []uuid.UUID
	f.handle = func(_ context.Context, m outbox.Message) error {
		calls = append(calls, m.Attempts)
		if len(calls) < 3 {
			return errors.New("smtp: connection refused")
		}
		delivered = append(delivered, m.ID)
		return nil
	}
	m := f.add(t, "mail")

	f.drain(t) // fails, retry in 1s
	f.drain(t) // not due yet
	f.now = f.now.Add(outbox.Backoff(1))
	f.drain(t) // fails, retry in 2s
	f.now = f.now.Add(outbox.Backoff(2))
	f.drain(t) // succeeds
	f.now = f.now.Add(time.Hour)
	f.drain(t) // nothing left

	if !slices.Equal(calls, []int{1, 2, 3}) {
		t.Errorf("handler saw attempts %v, want [1 2 3]", calls)
	}
	if !slices.Equal(delivered, []uuid.UUID{m.ID}) {
		t.Errorf("delivered %v, want exactly %s", delivered, m.ID)
	}
}

func TestRelayBuriesAfterMaxAttempts(t *testing.T) {
	f := newFixture(t)
	calls := 0
	f.handle = func(context.Context, outbox.Message) error { calls++; return errors.New("mailbox full") }
	m := f.add(t, "mail")

	for range 10 {
		f.drain(t)
		f.now = f.now.Add(time.Hour)
	}
	if calls != 3 {
		t.Errorf("handler called %d times, want 3", calls)
	}
	if got := f.store.Buried(); !slices.Equal(got, []uuid.UUID{m.ID}) {
		t.Errorf("buried %v, want %s", got, m.ID)
	}
}

func TestRelayBuriesUnknownTopics(t *testing.T) {
	f := newFixture(t)
	m := f.add(t, "sms")
	f.drain(t)
	if got := f.store.Buried(); !slices.Equal(got, []uuid.UUID{m.ID}) {
		t.Errorf("buried %v, want %s", got, m.ID)
	}
}

func TestRelaySurvivesPanickingHandler(t *testing.T) {
	f := newFixture(t)
	calls := 0
	f.handle = func(context.Context, outbox.Message) error {
		calls++
		if calls == 1 {
			panic("template is nil")
		}
		return nil
	}
	f.add(t, "mail")

	f.drain(t)
	f.now = f.now.Add(time.Hour)
	f.drain(t)
	if calls != 2 {
		t.Errorf("handler called %d times, want a retry after the panic", calls)
	}
}

func TestRelayBuriesMessagesThatCrashTheRelay(t *testing.T) {
	f := newFixture(t)
	m := f.add(t, "mail")
	// Three relays claimed it and died before finishing; their leases ran out.
	for range 3 {
		if _, err := f.store.Claim(t.Context(), f.now, time.Minute, 10); err != nil {
			t.Fatal(err)
		}
		f.now = f.now.Add(time.Hour)
	}

	f.handle = func(context.Context, outbox.Message) error {
		t.Error("handler ran for a message claimed too often")
		return nil
	}
	f.drain(t)
	if got := f.store.Buried(); !slices.Equal(got, []uuid.UUID{m.ID}) {
		t.Errorf("buried %v, want %s", got, m.ID)
	}
}

func TestRelayDeliversABatchOnItsWorkers(t *testing.T) {
	for _, tc := range []struct {
		workers int
		want    int32
	}{{workers: 2, want: 2}, {workers: 5, want: 3}} {
		synctest.Test(t, func(t *testing.T) {
			f := newFixture(t)
			f.relay.Config.Workers = tc.workers
			var running atomic.Int32
			release := make(chan struct{})
			f.handle = func(context.Context, outbox.Message) error {
				running.Add(1)
				<-release
				return nil
			}
			for range 3 {
				f.add(t, "mail")
			}

			done := make(chan int)
			go func() {
				n, err := f.relay.Drain(t.Context())
				if err != nil {
					t.Error(err)
				}
				done <- n
			}()
			synctest.Wait()
			if got := running.Load(); got != tc.want {
				t.Errorf("%d workers: %d deliveries running at once, want %d", tc.workers, got, tc.want)
			}
			close(release)
			if n := <-done; n != 3 {
				t.Errorf("Drain handled %d, want 3", n)
			}
		})
	}
}

func TestRelayRunPollsUntilCancelled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		db := memdb.New()
		store := memdb.NewOutbox(db)
		var delivered atomic.Int32
		relay := &outbox.Relay{
			Store:    store,
			Handlers: map[string]outbox.Handler{"mail": func(context.Context, outbox.Message) error { delivered.Add(1); return nil }},
			Config:   outbox.RelayConfig{Interval: time.Second},
			Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		}

		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error)
		go func() { done <- relay.Run(ctx) }()

		time.Sleep(500 * time.Millisecond)
		m, _ := outbox.NewMessage("mail", struct{}{}, time.Now())
		if err := db.Do(ctx, func(tx uow.Tx) error { return store.Add(ctx, tx, m) }); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Second)
		synctest.Wait()
		if delivered.Load() != 1 {
			t.Errorf("delivered %d messages after one poll, want 1", delivered.Load())
		}

		cancel()
		if err := <-done; err != nil {
			t.Errorf("Run returned %v after cancel, want nil", err)
		}
	})
}

func TestBackoff(t *testing.T) {
	for attempt, want := range map[int]time.Duration{0: time.Second, 1: time.Second, 2: 2 * time.Second, 4: 8 * time.Second, 9: 256 * time.Second, 10: 5 * time.Minute, 64: 5 * time.Minute} {
		if got := outbox.Backoff(attempt); got != want {
			t.Errorf("Backoff(%d) = %s, want %s", attempt, got, want)
		}
	}
}
