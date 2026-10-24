package mongodb_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/vardius/go-api-boilerplate/internal/platform/config"
	"github.com/vardius/go-api-boilerplate/internal/platform/mongodb"
	"github.com/vardius/go-api-boilerplate/internal/platform/mongodb/mongotest"
	"github.com/vardius/go-api-boilerplate/internal/platform/outbox"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

var t0 = time.Date(2026, 10, 24, 9, 30, 15, 123_000_000, time.UTC)

type otherTx struct{}

func (otherTx) Backend() string { return "mysql" }

func TestConnectFails(t *testing.T) {
	tests := []struct {
		name, uri string
	}{
		{"malformed URI", "http://localhost:27017"},
		{"no server", "mongodb://127.0.0.1:1/?serverSelectionTimeoutMS=100&connectTimeoutMS=100"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := mongodb.Connect(t.Context(), mongodb.Config{URI: config.Secret(tt.uri), Database: "test"})
			if err == nil {
				_ = db.Client.Disconnect(t.Context())
				t.Fatal("Connect succeeded")
			}
		})
	}
}

func TestSessionContextRejectsForeignTx(t *testing.T) {
	for _, tx := range []uow.Tx{otherTx{}, nil} {
		if _, err := mongodb.SessionContext(t.Context(), tx); err == nil {
			t.Errorf("SessionContext accepted a %T", tx)
		}
	}
}

func TestSessionContextRejectsFinishedTx(t *testing.T) {
	db := mongotest.New(t)
	var leaked uow.Tx
	if err := db.Do(t.Context(), func(tx uow.Tx) error { leaked = tx; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := mongodb.SessionContext(t.Context(), leaked); !errors.Is(err, mongodb.ErrTxDone) {
		t.Errorf("SessionContext on a finished transaction: err = %v, want ErrTxDone", err)
	}
}

func TestStoresRejectForeignTx(t *testing.T) {
	db := mongotest.New(t)
	if err := mongodb.NewEventStore(db).Append(t.Context(), otherTx{}, uuid.New(), 0, nil); err == nil {
		t.Error("EventStore.Append accepted another backend's transaction")
	}
	if err := mongodb.NewOutbox(db).Add(t.Context(), otherTx{}); err == nil {
		t.Error("Outbox.Add accepted another backend's transaction")
	}
}

func TestDoReturnsFunctionErrorUnchanged(t *testing.T) {
	db := mongotest.New(t)
	failure := errors.New("projection failed")
	if err := db.Do(t.Context(), func(uow.Tx) error { return failure }); err != failure {
		t.Errorf("Do returned %v, want the function's error itself", err)
	}
}

func TestDoRollsBackOnPanic(t *testing.T) {
	db := mongotest.New(t)
	store := mongodb.NewOutbox(db)
	m, err := outbox.NewMessage("topic", struct{}{}, t0)
	if err != nil {
		t.Fatal(err)
	}

	recovered := func() (v any) {
		defer func() { v = recover() }()
		_ = db.Do(t.Context(), func(tx uow.Tx) error {
			if err := store.Add(t.Context(), tx, m); err != nil {
				return err
			}
			panic("boom")
		})
		return nil
	}()
	if recovered != "boom" {
		t.Fatalf("recovered %v, want the function's panic", recovered)
	}

	got, err := store.Claim(t.Context(), t0, time.Minute, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("a panicking transaction kept %d messages", len(got))
	}
}

func TestDoDoesNotCommitAfterContextEnds(t *testing.T) {
	db := mongotest.New(t)
	store := mongodb.NewOutbox(db)
	m, err := outbox.NewMessage("topic", struct{}{}, t0)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	err = db.Do(ctx, func(tx uow.Tx) error {
		if err := store.Add(ctx, tx, m); err != nil {
			return err
		}
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Do returned %v, want context.Canceled", err)
	}

	got, err := store.Claim(t.Context(), t0, time.Minute, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("a transaction whose context ended kept %d messages", len(got))
	}
}

// Service stores write read models through SessionContext, and a read
// model's first write may create its collection.
func TestDoWritesToNewCollection(t *testing.T) {
	db := mongotest.New(t)
	coll := db.Database.Collection("projection")
	err := db.Do(t.Context(), func(tx uow.Tx) error {
		ctx, err := mongodb.SessionContext(t.Context(), tx)
		if err != nil {
			return err
		}
		_, err = coll.InsertOne(ctx, bson.D{{Key: "_id", Value: 1}})
		return err
	})
	if err != nil {
		t.Fatalf("insert into a new collection in a transaction: %v", err)
	}
	if n, err := coll.CountDocuments(t.Context(), bson.D{}); err != nil || n != 1 {
		t.Fatalf("collection holds %d documents, %v; want 1", n, err)
	}
}

func TestIsDuplicate(t *testing.T) {
	db := mongotest.New(t)
	coll := db.Database.Collection("things")
	if _, err := coll.InsertOne(t.Context(), bson.D{{Key: "_id", Value: 1}}); err != nil {
		t.Fatal(err)
	}
	_, dup := coll.InsertOne(t.Context(), bson.D{{Key: "_id", Value: 1}})
	_, other := coll.InsertOne(t.Context(), bson.D{{Key: "_id", Value: bson.A{1}}})

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"duplicate key", dup, true},
		{"wrapped duplicate key", fmt.Errorf("save thing: %w", dup), true},
		{"other server error", other, false},
		{"plain error", errors.New("duplicate key"), false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mongodb.IsDuplicate(tt.err); got != tt.want {
				t.Errorf("IsDuplicate(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
	if other == nil {
		t.Error("inserting an array _id succeeded; the test needs another non-duplicate error")
	}
}

// indexes returns a collection's indexes by name, each as its key names in
// order, and checks that every key is ascending, that only the events index
// on stream and version is unique, and that none is a TTL index.
func indexes(t *testing.T, coll *mongo.Collection) map[string][]string {
	t.Helper()
	specs, err := coll.Indexes().ListSpecifications(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	for _, s := range specs {
		elems, err := s.KeysDocument.Elements()
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range elems {
			if n, ok := e.Value().AsInt64OK(); !ok || n != 1 {
				t.Errorf("%s: index %s orders %s by %s, want 1", coll.Name(), s.Name, e.Key(), e.Value())
			}
			got[s.Name] = append(got[s.Name], e.Key())
		}
		if s.ExpireAfterSeconds != nil {
			t.Errorf("%s: index %s is a TTL index", coll.Name(), s.Name)
		}
		if unique := s.Unique != nil && *s.Unique; unique != (s.Name == "stream_id_1_version_1") {
			t.Errorf("%s: index %s unique = %v", coll.Name(), s.Name, unique)
		}
	}
	return got
}

func TestEnsureIndexes(t *testing.T) {
	db := mongotest.New(t) // has ensured them once

	if err := mongodb.EnsureIndexes(t.Context(), db.Database, mongodb.Indexes...); err != nil {
		t.Fatalf("ensuring indexes again: %v", err)
	}
	want := map[string]map[string][]string{
		"events": {
			"_id_":                  {"_id"},
			"stream_id_1_version_1": {"stream_id", "version"},
		},
		"outbox": {
			"_id_":                       {"_id"},
			"buried_at_1_due_at_1__id_1": {"buried_at", "due_at", "_id"},
		},
	}
	for coll, want := range want {
		if got := indexes(t, db.Database.Collection(coll)); !maps.EqualFunc(got, want, slices.Equal) {
			t.Errorf("%s indexes = %v, want %v", coll, got, want)
		}
	}

	conflicting := mongodb.Index{Collection: "events", Model: mongo.IndexModel{
		Keys: bson.D{{Key: "stream_id", Value: 1}, {Key: "version", Value: 1}},
	}}
	if err := mongodb.EnsureIndexes(t.Context(), db.Database, conflicting); err == nil {
		t.Error("EnsureIndexes accepted an index that differs from the existing one of the same name")
	}
}

func TestOperationsFailOnEndedContext(t *testing.T) {
	db := mongotest.New(t)
	events, msgs := mongodb.NewEventStore(db), mongodb.NewOutbox(db)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	id := uuid.New()

	ops := map[string]func() error{
		"Load":  func() error { _, err := events.Load(ctx, id); return err },
		"Claim": func() error { _, err := msgs.Claim(ctx, t0, time.Minute, 1); return err },
		"Ack":   func() error { return msgs.Ack(ctx, id) },
		"Retry": func() error { return msgs.Retry(ctx, id, t0, "x") },
		"Bury":  func() error { return msgs.Bury(ctx, id, "x") },
		"EnsureIndexes": func() error {
			return mongodb.EnsureIndexes(ctx, db.Database, mongodb.Indexes...)
		},
		"Do": func() error {
			return db.Do(ctx, func(tx uow.Tx) error { return events.Append(ctx, tx, id, 0, nil) })
		},
	}
	for name, op := range ops {
		if err := op(); !errors.Is(err, context.Canceled) {
			t.Errorf("%s on an ended context: err = %v, want context.Canceled", name, err)
		}
	}
}

// rawDoc reads the document with the given _id, a UUID, as stored.
func rawDoc(t *testing.T, coll *mongo.Collection, id uuid.UUID) bson.Raw {
	t.Helper()
	raw, err := coll.FindOne(t.Context(), bson.D{{Key: "_id", Value: bson.Binary{Subtype: 0x04, Data: id[:]}}}).Raw()
	if err != nil {
		t.Fatalf("read %s %s: %v", coll.Name(), id, err)
	}
	return raw
}
