package memdb_test

import (
	"errors"
	"testing"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/platform/es"
	"github.com/vardius/go-api-boilerplate/internal/platform/es/estest"
	"github.com/vardius/go-api-boilerplate/internal/platform/memdb"
	"github.com/vardius/go-api-boilerplate/internal/platform/outbox"
	"github.com/vardius/go-api-boilerplate/internal/platform/outbox/outboxtest"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

func TestEventStore(t *testing.T) {
	estest.TestStore(t, func(*testing.T) (es.Store, uow.UnitOfWork) {
		db := memdb.New()
		return memdb.NewEventStore(db), db
	})
}

func TestOutbox(t *testing.T) {
	outboxtest.TestStore(t, func(*testing.T) (outbox.Store, uow.UnitOfWork) {
		db := memdb.New()
		return memdb.NewOutbox(db), db
	})
}

type otherTx struct{}

func (otherTx) Backend() string { return "mysql" }

func TestFromRejectsForeignAndFinishedTx(t *testing.T) {
	if _, err := memdb.From(otherTx{}); err == nil {
		t.Error("From accepted another backend's transaction")
	}

	var leaked uow.Tx
	db := memdb.New()
	if err := db.Do(t.Context(), func(tx uow.Tx) error { leaked = tx; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := memdb.From(leaked); !errors.Is(err, memdb.ErrTxDone) {
		t.Errorf("From on a finished transaction: err = %v, want ErrTxDone", err)
	}
}

func TestPanicRollsBack(t *testing.T) {
	db := memdb.New()
	store := memdb.NewOutbox(db)
	m, err := outbox.NewMessage("topic", struct{}{}, t0)
	if err != nil {
		t.Fatal(err)
	}

	func() {
		defer func() { _ = recover() }()
		_ = db.Do(t.Context(), func(tx uow.Tx) error {
			if err := store.Add(t.Context(), tx, m); err != nil {
				return err
			}
			panic("boom")
		})
	}()

	got, err := store.Claim(t.Context(), t0, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("a panicking transaction kept %d messages", len(got))
	}
}

var t0 = mustTime("2026-10-24T09:00:00Z")

func mustTime(s string) time.Time {
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return v
}
