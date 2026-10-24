package estest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/es"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// OpenStore returns an empty store and the unit of work its transactions
// come from. Each call must return a store that shares no data with others.
type OpenStore func(t *testing.T) (es.Store, uow.UnitOfWork)

// TestStore checks the rules every es.Store must follow. Each backend's
// tests call it:
//
//	func TestEventStore(t *testing.T) {
//		estest.TestStore(t, func(t *testing.T) (es.Store, uow.UnitOfWork) {
//			db := memdb.New()
//			return memdb.NewEventStore(db), db
//		})
//	}
func TestStore(t *testing.T, open OpenStore) {
	t.Run("AppendThenLoad", func(t *testing.T) { testAppendThenLoad(t, open) })
	t.Run("AppendContinuesStream", func(t *testing.T) { testAppendContinuesStream(t, open) })
	t.Run("StaleVersionConflicts", func(t *testing.T) { testVersionConflicts(t, open, 1) })
	t.Run("FutureVersionConflicts", func(t *testing.T) { testVersionConflicts(t, open, 3) })
	t.Run("ConcurrentAppendsOneWins", func(t *testing.T) { testConcurrentAppends(t, open) })
	t.Run("RollbackDiscardsAppend", func(t *testing.T) { testRollback(t, open) })
	t.Run("UnknownStreamIsEmpty", func(t *testing.T) { testUnknownStream(t, open) })
	t.Run("StreamsAreIndependent", func(t *testing.T) { testIndependentStreams(t, open) })
}

// at is a fixed event time with the millisecond precision stores keep.
var at = time.Date(2026, 10, 24, 9, 30, 15, 123_000_000, time.UTC)

// records returns n records for stream, with versions from+1 up.
func records(stream uuid.UUID, from, n int) []es.Record {
	recs := make([]es.Record, n)
	for i := range recs {
		v := from + i + 1
		recs[i] = es.Record{
			ID:         uuid.NewV7(),
			StreamID:   stream,
			StreamType: "test",
			Version:    v,
			Type:       "test.happened.v1",
			OccurredAt: at.Add(time.Duration(v) * time.Second),
			Payload:    fmt.Appendf(nil, `{"n":%d,"text":"zażółć ✓","nested":{"list":[1,2,3]}}`, v),
			Meta:       es.Meta{CorrelationID: "corr", CausationID: "cause", ActorID: fmt.Sprint("actor-", v)},
		}
	}
	return recs
}

func appendIn(ctx context.Context, store es.Store, u uow.UnitOfWork, stream uuid.UUID, expected int, recs []es.Record) error {
	return u.Do(ctx, func(tx uow.Tx) error {
		return store.Append(ctx, tx, stream, expected, recs)
	})
}

func mustAppend(t *testing.T, store es.Store, u uow.UnitOfWork, stream uuid.UUID, expected int, recs []es.Record) {
	t.Helper()
	if err := appendIn(t.Context(), store, u, stream, expected, recs); err != nil {
		t.Fatalf("append at version %d: %v", expected, err)
	}
}

func mustLoad(t *testing.T, store es.Store, stream uuid.UUID) []es.Record {
	t.Helper()
	recs, err := store.Load(t.Context(), stream)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return recs
}

// assertRecords compares records field by field. Payloads are compared as
// JSON values, since a store may reformat them (MySQL's JSON type does).
func assertRecords(t *testing.T, got, want []es.Record) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("loaded %d records, want %d", len(got), len(want))
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.ID != w.ID || g.StreamID != w.StreamID || g.StreamType != w.StreamType ||
			g.Version != w.Version || g.Type != w.Type || g.Meta != w.Meta {
			t.Errorf("record %d:\n got %+v\nwant %+v", i, g, w)
		}
		if !g.OccurredAt.Equal(w.OccurredAt) {
			t.Errorf("record %d: occurred at %s, want %s", i, g.OccurredAt, w.OccurredAt)
		}
		if !jsonEqual(t, g.Payload, w.Payload) {
			t.Errorf("record %d: payload %s, want %s", i, g.Payload, w.Payload)
		}
	}
}

func jsonEqual(t *testing.T, a, b []byte) bool {
	t.Helper()
	var x, y any
	if err := json.Unmarshal(a, &x); err != nil {
		t.Errorf("payload %q is not JSON: %v", a, err)
		return false
	}
	if err := json.Unmarshal(b, &y); err != nil {
		t.Errorf("payload %q is not JSON: %v", b, err)
		return false
	}
	return reflect.DeepEqual(x, y)
}

func testAppendThenLoad(t *testing.T, open OpenStore) {
	store, u := open(t)
	stream := uuid.New()
	want := records(stream, 0, 3)

	mustAppend(t, store, u, stream, 0, want)
	assertRecords(t, mustLoad(t, store, stream), want)
}

func testAppendContinuesStream(t *testing.T, open OpenStore) {
	store, u := open(t)
	stream := uuid.New()
	first, second := records(stream, 0, 2), records(stream, 2, 2)

	mustAppend(t, store, u, stream, 0, first)
	mustAppend(t, store, u, stream, 2, second)
	assertRecords(t, mustLoad(t, store, stream), append(first, second...))
}

func testVersionConflicts(t *testing.T, open OpenStore, expected int) {
	store, u := open(t)
	stream := uuid.New()
	stored := records(stream, 0, 2)
	mustAppend(t, store, u, stream, 0, stored)

	err := appendIn(t.Context(), store, u, stream, expected, records(stream, expected, 1))
	if !errors.Is(err, es.ErrConcurrency) {
		t.Fatalf("append at version %d to a stream at version 2: err = %v, want es.ErrConcurrency", expected, err)
	}
	assertRecords(t, mustLoad(t, store, stream), stored)
}

func testConcurrentAppends(t *testing.T, open OpenStore) {
	store, u := open(t)
	stream := uuid.New()
	mustAppend(t, store, u, stream, 0, records(stream, 0, 1))

	const writers = 8
	errs := make([]error, writers)
	var wg sync.WaitGroup
	for i := range writers {
		wg.Go(func() {
			errs[i] = appendIn(t.Context(), store, u, stream, 1, records(stream, 1, 2))
		})
	}
	wg.Wait()

	won := 0
	for i, err := range errs {
		switch {
		case err == nil:
			won++
		case !errors.Is(err, es.ErrConcurrency):
			t.Errorf("writer %d: err = %v, want nil or es.ErrConcurrency", i, err)
		}
	}
	if won != 1 {
		t.Fatalf("%d of %d concurrent appends at the same version succeeded, want exactly 1", won, writers)
	}
	if got := mustLoad(t, store, stream); len(got) != 3 {
		t.Fatalf("stream has %d records, want 3", len(got))
	}
}

func testRollback(t *testing.T, open OpenStore) {
	store, u := open(t)
	stream := uuid.New()
	kept := records(stream, 0, 1)
	mustAppend(t, store, u, stream, 0, kept)

	failure := errors.New("projection failed")
	err := u.Do(t.Context(), func(tx uow.Tx) error {
		if err := store.Append(t.Context(), tx, stream, 1, records(stream, 1, 2)); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("Do returned %v, want the function's error", err)
	}
	assertRecords(t, mustLoad(t, store, stream), kept)

	// The rolled-back versions are free again.
	mustAppend(t, store, u, stream, 1, records(stream, 1, 1))
}

func testUnknownStream(t *testing.T, open OpenStore) {
	store, _ := open(t)
	if got := mustLoad(t, store, uuid.New()); len(got) != 0 {
		t.Fatalf("an unknown stream loaded %d records, want none", len(got))
	}
}

func testIndependentStreams(t *testing.T, open OpenStore) {
	store, u := open(t)
	a, b := uuid.New(), uuid.New()
	recsA, recsB := records(a, 0, 2), records(b, 0, 1)

	mustAppend(t, store, u, a, 0, recsA)
	mustAppend(t, store, u, b, 0, recsB)
	assertRecords(t, mustLoad(t, store, a), recsA)
	assertRecords(t, mustLoad(t, store, b), recsB)
}
