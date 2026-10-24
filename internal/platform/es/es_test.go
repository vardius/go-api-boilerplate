package es_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/es"
	"github.com/vardius/go-api-boilerplate/internal/platform/es/estest"
	"github.com/vardius/go-api-boilerplate/internal/platform/memdb"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// counter is the smallest aggregate: it opens, then counts up to a limit.
type counter struct {
	es.Changes
	id    uuid.UUID
	count int
}

type opened struct {
	ID uuid.UUID `json:"id"`
}

type incremented struct {
	By int `json:"by"`
}

var errLimit = errors.New("limit reached")

func (c *counter) ID() uuid.UUID { return c.id }

func (c *counter) Apply(event any) error {
	switch e := event.(type) {
	case opened:
		c.id = e.ID
	case incremented:
		c.count += e.By
	default:
		return fmt.Errorf("counter: can't apply %T", event)
	}
	return nil
}

func (c *counter) increment(by int) error {
	if c.count+by > 10 {
		return errLimit
	}
	return es.Raise(c, incremented{By: by})
}

func registry() *es.Registry {
	r := es.NewRegistry()
	es.Register[opened](r, "counter.opened.v1")
	es.Register[incremented](r, "counter.incremented.v1")
	return r
}

func TestScenario(t *testing.T) {
	id := uuid.New()
	estest.Given(t, &counter{}, opened{ID: id}, incremented{By: 4}).
		When(func(c *counter) error { return c.increment(6) }).
		Then(incremented{By: 6})

	estest.Given(t, &counter{}, opened{ID: id}, incremented{By: 4}).
		When(func(c *counter) error { return c.increment(7) }).
		ThenError(errLimit)
}

func TestRegistryRoundTrip(t *testing.T) {
	r := registry()
	name, data, err := r.Encode(incremented{By: 3})
	if err != nil {
		t.Fatal(err)
	}
	if name != "counter.incremented.v1" {
		t.Errorf("name = %q", name)
	}
	got, err := r.Decode(name, data)
	if err != nil {
		t.Fatal(err)
	}
	if got != (incremented{By: 3}) {
		t.Errorf("decoded %#v", got)
	}
}

func TestRegistryRejectsUnknownTypes(t *testing.T) {
	r := registry()
	if _, err := r.Decode("counter.reset.v1", []byte(`{}`)); !errors.Is(err, es.ErrUnknownEventType) {
		t.Errorf("decode unknown type: err = %v, want ErrUnknownEventType", err)
	}
	if _, _, err := r.Encode(struct{ X int }{}); err == nil {
		t.Error("encoded an unregistered type")
	}
	if _, err := r.Decode("counter.incremented.v1", []byte(`{"by":"three"}`)); err == nil {
		t.Error("decoded a malformed payload")
	}
}

func TestRegistryPanicsOnMistakes(t *testing.T) {
	tests := map[string]func(r *es.Registry){
		"empty name":     func(r *es.Registry) { es.Register[opened](es.NewRegistry(), "") },
		"duplicate name": func(r *es.Registry) { es.Register[struct{ A int }](r, "counter.opened.v1") },
		"duplicate type": func(r *es.Registry) { es.Register[opened](r, "counter.opened.v2") },
		"pointer type":   func(r *es.Registry) { es.Register[*opened](r, "counter.pointer.v1") },
		"upcast a registered type": func(r *es.Registry) {
			r.Upcast("counter.opened.v1", "counter.opened.v2", nil)
		},
	}
	for name, register := range tests {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("did not panic")
				}
			}()
			register(registry())
		})
	}
}

func TestUpcast(t *testing.T) {
	r := registry()
	// v1 of incremented stored the step as "step"; v2 renamed it, and v3 is the current shape.
	r.Upcast("counter.incremented.v0", "counter.incremented.v00", func(old []byte) ([]byte, error) {
		return []byte(strings.Replace(string(old), `"step"`, `"by"`, 1)), nil
	})
	r.Upcast("counter.incremented.v00", "counter.incremented.v1", func(old []byte) ([]byte, error) {
		var v map[string]int
		if err := json.Unmarshal(old, &v); err != nil {
			return nil, err
		}
		return json.Marshal(map[string]int{"by": v["by"] * 2})
	})

	got, err := r.Decode("counter.incremented.v0", []byte(`{"step":2}`))
	if err != nil {
		t.Fatal(err)
	}
	if got != (incremented{By: 4}) {
		t.Errorf("decoded %#v, want by 4", got)
	}
}

func newRepository(project es.Projection) (*es.Repository[*counter], *memdb.DB, *memdb.EventStore) {
	db := memdb.New()
	store := memdb.NewEventStore(db)
	at := time.Date(2026, 10, 24, 9, 0, 0, 123_456_789, time.UTC)
	return es.NewRepository(es.RepositoryConfig[*counter]{
		StreamType: "counter",
		New:        func() *counter { return &counter{} },
		Store:      store,
		Registry:   registry(),
		Project:    project,
		Now:        func() time.Time { return at },
	}), db, store
}

func save(ctx context.Context, db uow.UnitOfWork, repo *es.Repository[*counter], c *counter) ([]es.Event, error) {
	var events []es.Event
	err := db.Do(ctx, func(tx uow.Tx) error {
		var err error
		events, err = repo.Save(ctx, tx, c, es.Meta{ActorID: "tester"})
		return err
	})
	return events, err
}

func TestRepositorySaveAndLoad(t *testing.T) {
	var projected []es.Event
	repo, db, _ := newRepository(func(_ context.Context, _ uow.Tx, events []es.Event) error {
		projected = append(projected, events...)
		return nil
	})

	c := &counter{}
	id := uuid.New()
	if err := es.Raise(c, opened{ID: id}); err != nil {
		t.Fatal(err)
	}
	if err := c.increment(2); err != nil {
		t.Fatal(err)
	}
	events, err := save(t.Context(), db, repo, c)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Version != 1 || events[1].Version != 2 || events[1].Type != "counter.incremented.v1" {
		t.Fatalf("saved %+v", events)
	}
	if events[0].Meta.ActorID != "tester" || events[0].OccurredAt.Nanosecond()%int(time.Millisecond) != 0 {
		t.Errorf("event 0 = %+v: want the actor and a millisecond timestamp", events[0])
	}
	if len(projected) != 2 {
		t.Errorf("projected %d events, want 2", len(projected))
	}

	loaded, err := repo.Load(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.count != 2 || loaded.Version() != 2 || len(es.Pending(loaded)) != 0 {
		t.Errorf("loaded count %d, version %d, pending %d", loaded.count, loaded.Version(), len(es.Pending(loaded)))
	}

	if err := loaded.increment(1); err != nil {
		t.Fatal(err)
	}
	if _, err := save(t.Context(), db, repo, loaded); err != nil {
		t.Fatal(err)
	}
	if again, _ := repo.Load(t.Context(), id); again.Version() != 3 {
		t.Errorf("version after a second save = %d, want 3", again.Version())
	}
}

func TestRepositoryConflict(t *testing.T) {
	repo, db, _ := newRepository(nil)
	id := uuid.New()
	c := &counter{}
	_ = es.Raise(c, opened{ID: id})
	if _, err := save(t.Context(), db, repo, c); err != nil {
		t.Fatal(err)
	}

	first, _ := repo.Load(t.Context(), id)
	second, _ := repo.Load(t.Context(), id)
	_ = first.increment(1)
	_ = second.increment(1)
	if _, err := save(t.Context(), db, repo, first); err != nil {
		t.Fatal(err)
	}
	_, err := save(t.Context(), db, repo, second)
	if !errors.Is(err, es.ErrConcurrency) || !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("second writer: err = %v, want ErrConcurrency, an ErrConflict", err)
	}
}

func TestRepositoryProjectionFailureRollsBack(t *testing.T) {
	failure := errors.New("email taken")
	repo, db, store := newRepository(func(context.Context, uow.Tx, []es.Event) error { return failure })
	c := &counter{}
	id := uuid.New()
	_ = es.Raise(c, opened{ID: id})

	if _, err := save(t.Context(), db, repo, c); !errors.Is(err, failure) {
		t.Fatalf("err = %v, want the projection's error", err)
	}
	if recs, _ := store.Load(t.Context(), id); len(recs) != 0 {
		t.Fatalf("a failed projection kept %d events", len(recs))
	}
}

func TestRepositoryLoadErrors(t *testing.T) {
	repo, db, store := newRepository(nil)
	if _, err := repo.Load(t.Context(), uuid.New()); !errors.Is(err, apperr.ErrNotFound) {
		t.Errorf("load a missing stream: err = %v, want ErrNotFound", err)
	}

	id := uuid.New()
	err := db.Do(t.Context(), func(tx uow.Tx) error {
		return store.Append(t.Context(), tx, id, 0, []es.Record{
			{ID: uuid.NewV7(), StreamID: id, StreamType: "counter", Version: 1, Type: "counter.exploded.v1", Payload: []byte(`{}`)},
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Load(t.Context(), id); !errors.Is(err, es.ErrUnknownEventType) {
		t.Errorf("load a stream with an unknown event: err = %v, want ErrUnknownEventType", err)
	}

	other := uuid.New()
	_ = db.Do(t.Context(), func(tx uow.Tx) error {
		return store.Append(t.Context(), tx, other, 0, []es.Record{
			{ID: uuid.NewV7(), StreamID: other, StreamType: "invoice", Version: 1, Type: "counter.opened.v1", Payload: []byte(`{}`)},
		})
	})
	if _, err := repo.Load(t.Context(), other); err == nil || !strings.Contains(err.Error(), "invoice") {
		t.Errorf("load another stream type: err = %v", err)
	}
}

func TestSaveWithNothingPendingWritesNothing(t *testing.T) {
	repo, db, _ := newRepository(func(context.Context, uow.Tx, []es.Event) error {
		t.Error("projection ran with nothing to save")
		return nil
	})
	events, err := save(t.Context(), db, repo, &counter{id: uuid.New()})
	if err != nil || events != nil {
		t.Fatalf("save = %v, %v; want nothing", events, err)
	}
}
