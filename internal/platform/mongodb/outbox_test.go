package mongodb_test

import (
	"errors"
	"testing"
	"time"
	"uuid"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/mongodb"
	"github.com/vardius/go-api-boilerplate/internal/platform/mongodb/mongotest"
	"github.com/vardius/go-api-boilerplate/internal/platform/outbox"
	"github.com/vardius/go-api-boilerplate/internal/platform/outbox/outboxtest"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

func TestOutbox(t *testing.T) {
	outboxtest.TestStore(t, func(t *testing.T) (outbox.Store, uow.UnitOfWork) {
		db := mongotest.New(t)
		return mongodb.NewOutbox(db), db
	})
}

func addMessages(t *testing.T, db *mongodb.DB, msgs ...outbox.Message) error {
	t.Helper()
	store := mongodb.NewOutbox(db)
	return db.Do(t.Context(), func(tx uow.Tx) error { return store.Add(t.Context(), tx, msgs...) })
}

func newMessage(t *testing.T) outbox.Message {
	t.Helper()
	m, err := outbox.NewMessage("mail.login_link", map[string]string{"to": "a@example.com"}, t0)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// messageFields reads a stored message's fields that change over its life.
type messageFields struct {
	ID           bson.RawValue `bson:"_id"`
	Topic        string        `bson:"topic"`
	Payload      bson.RawValue `bson:"payload"`
	CreatedAt    bson.RawValue `bson:"created_at"`
	DueAt        bson.RawValue `bson:"due_at"`
	Attempts     bson.RawValue `bson:"attempts"`
	ClaimedUntil bson.RawValue `bson:"claimed_until"`
	BuriedAt     bson.RawValue `bson:"buried_at"`
	LastError    bson.RawValue `bson:"last_error"`
}

func readMessage(t *testing.T, db *mongodb.DB, id uuid.UUID) messageFields {
	t.Helper()
	var f messageFields
	if err := bson.Unmarshal(rawDoc(t, db.Database.Collection("outbox"), id), &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func assertDate(t *testing.T, field string, v bson.RawValue, want time.Time) {
	t.Helper()
	if ms, ok := v.DateTimeOK(); !ok || ms != want.UnixMilli() {
		t.Errorf("%s = %s, want the date %s", field, v, want)
	}
}

func assertAbsent(t *testing.T, field string, v bson.RawValue) {
	t.Helper()
	if v.Type != 0 {
		t.Errorf("%s = %s, want it absent", field, v)
	}
}

func TestOutboxDocumentShape(t *testing.T) {
	db := mongotest.New(t)
	store := mongodb.NewOutbox(db)
	m := newMessage(t)
	if err := addMessages(t, db, m); err != nil {
		t.Fatal(err)
	}

	added := readMessage(t, db, m.ID)
	assertUUID(t, "_id", added.ID, m.ID)
	if added.Topic != m.Topic {
		t.Errorf("topic = %q, want %q", added.Topic, m.Topic)
	}
	if s, ok := added.Payload.StringValueOK(); !ok || s != string(m.Payload) {
		t.Errorf("payload = %s, want the string %s", added.Payload, m.Payload)
	}
	assertDate(t, "created_at", added.CreatedAt, t0)
	assertDate(t, "due_at", added.DueAt, t0)
	if n, ok := added.Attempts.AsInt64OK(); !ok || n != 0 {
		t.Errorf("attempts = %s, want 0", added.Attempts)
	}
	assertAbsent(t, "claimed_until", added.ClaimedUntil)
	assertAbsent(t, "buried_at", added.BuriedAt)
	assertAbsent(t, "last_error", added.LastError)

	if _, err := store.Claim(t.Context(), t0, time.Minute, 1); err != nil {
		t.Fatal(err)
	}
	claimed := readMessage(t, db, m.ID)
	assertDate(t, "claimed_until", claimed.ClaimedUntil, t0.Add(time.Minute))
	if n, ok := claimed.Attempts.AsInt64OK(); !ok || n != 1 {
		t.Errorf("attempts = %s, want 1", claimed.Attempts)
	}

	retryAt := t0.Add(10 * time.Second)
	if err := store.Retry(t.Context(), m.ID, retryAt, "smtp: connection refused"); err != nil {
		t.Fatal(err)
	}
	retried := readMessage(t, db, m.ID)
	assertDate(t, "due_at", retried.DueAt, retryAt)
	assertAbsent(t, "claimed_until", retried.ClaimedUntil)
	if s, _ := retried.LastError.StringValueOK(); s != "smtp: connection refused" {
		t.Errorf("last_error = %s", retried.LastError)
	}

	// Bury stamps the server's clock, so the test checks the type, not the time.
	if err := store.Bury(t.Context(), m.ID, "template missing"); err != nil {
		t.Fatal(err)
	}
	buried := readMessage(t, db, m.ID)
	if buried.BuriedAt.Type != bson.TypeDateTime {
		t.Errorf("buried_at = %s, want a date", buried.BuriedAt)
	}
	if s, _ := buried.LastError.StringValueOK(); s != "template missing" {
		t.Errorf("last_error = %s", buried.LastError)
	}
}

func TestOutboxUnknownMessage(t *testing.T) {
	db := mongotest.New(t)
	store := mongodb.NewOutbox(db)
	id := uuid.New()

	if err := store.Ack(t.Context(), id); err != nil {
		t.Errorf("ack of an unknown message: %v", err)
	}
	if err := store.Retry(t.Context(), id, t0, "x"); !errors.Is(err, apperr.ErrNotFound) {
		t.Errorf("retry of an unknown message: err = %v, want apperr.ErrNotFound", err)
	}
	if err := store.Bury(t.Context(), id, "x"); !errors.Is(err, apperr.ErrNotFound) {
		t.Errorf("bury of an unknown message: err = %v, want apperr.ErrNotFound", err)
	}
}

func TestOutboxAddTwiceConflicts(t *testing.T) {
	db := mongotest.New(t)
	m := newMessage(t)
	if err := addMessages(t, db, m); err != nil {
		t.Fatal(err)
	}
	if err := addMessages(t, db, m); !errors.Is(err, apperr.ErrConflict) {
		t.Errorf("adding a message again: err = %v, want apperr.ErrConflict", err)
	}
}

func TestOutboxEdgeCases(t *testing.T) {
	db := mongotest.New(t)
	store := mongodb.NewOutbox(db)
	if err := addMessages(t, db); err != nil {
		t.Errorf("adding no messages: %v", err)
	}
	if err := addMessages(t, db, newMessage(t)); err != nil {
		t.Fatal(err)
	}
	for _, limit := range []int{0, -1} {
		if got, err := store.Claim(t.Context(), t0, time.Minute, limit); err != nil || len(got) != 0 {
			t.Errorf("claim with limit %d = %d messages, %v; want none", limit, len(got), err)
		}
	}
}

func TestClaimRejectsMalformedID(t *testing.T) {
	db := mongotest.New(t)
	_, err := db.Database.Collection("outbox").InsertOne(t.Context(), bson.D{
		{Key: "_id", Value: bson.Binary{Subtype: 0x00, Data: make([]byte, 16)}},
		{Key: "topic", Value: "t"},
		{Key: "payload", Value: "{}"},
		{Key: "created_at", Value: bson.NewDateTimeFromTime(t0)},
		{Key: "due_at", Value: bson.NewDateTimeFromTime(t0)},
		{Key: "attempts", Value: 0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := mongodb.NewOutbox(db).Claim(t.Context(), t0, time.Minute, 1); err == nil {
		t.Fatalf("claim of a message whose _id isn't a UUID = %v, want an error", got)
	}
}
