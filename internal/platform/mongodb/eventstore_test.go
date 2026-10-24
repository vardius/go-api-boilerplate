package mongodb_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/vardius/go-api-boilerplate/internal/platform/es"
	"github.com/vardius/go-api-boilerplate/internal/platform/es/estest"
	"github.com/vardius/go-api-boilerplate/internal/platform/mongodb"
	"github.com/vardius/go-api-boilerplate/internal/platform/mongodb/mongotest"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

func TestEventStore(t *testing.T) {
	estest.TestStore(t, func(t *testing.T) (es.Store, uow.UnitOfWork) {
		db := mongotest.New(t)
		return mongodb.NewEventStore(db), db
	})
}

func record(stream uuid.UUID, version int, payload string) es.Record {
	return es.Record{
		ID:         uuid.NewV7(),
		StreamID:   stream,
		StreamType: "user",
		Version:    version,
		Type:       "user.registered.v1",
		OccurredAt: t0.Add(time.Duration(version) * time.Second),
		Payload:    []byte(payload),
		Meta:       es.Meta{CorrelationID: "corr", ActorID: "actor"},
	}
}

func appendRecords(t *testing.T, db *mongodb.DB, stream uuid.UUID, expected int, recs ...es.Record) error {
	t.Helper()
	store := mongodb.NewEventStore(db)
	return db.Do(t.Context(), func(tx uow.Tx) error {
		return store.Append(t.Context(), tx, stream, expected, recs)
	})
}

func TestEventDocumentShape(t *testing.T) {
	db := mongotest.New(t)
	stream := uuid.New()
	rec := record(stream, 1, `{"email":"a@example.com","age":42,"tags":["x"],"address":{"city":"Kraków"}}`)
	if err := appendRecords(t, db, stream, 0, rec); err != nil {
		t.Fatal(err)
	}

	raw := rawDoc(t, db.Database.Collection("events"), rec.ID)
	var doc struct {
		ID         bson.RawValue `bson:"_id"`
		StreamID   bson.RawValue `bson:"stream_id"`
		StreamType string        `bson:"stream_type"`
		Version    bson.RawValue `bson:"version"`
		Type       string        `bson:"type"`
		OccurredAt bson.RawValue `bson:"occurred_at"`
		Payload    bson.RawValue `bson:"payload"`
		Meta       bson.D        `bson:"meta"`
	}
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}

	assertUUID(t, "_id", doc.ID, rec.ID)
	assertUUID(t, "stream_id", doc.StreamID, stream)
	if doc.StreamType != "user" || doc.Type != "user.registered.v1" {
		t.Errorf("stream_type, type = %q, %q", doc.StreamType, doc.Type)
	}
	if v, ok := doc.Version.AsInt64OK(); !ok || v != 1 {
		t.Errorf("version = %s, want the number 1", doc.Version)
	}
	if ms, ok := doc.OccurredAt.DateTimeOK(); !ok || ms != rec.OccurredAt.UnixMilli() {
		t.Errorf("occurred_at = %s, want the date %s", doc.OccurredAt, rec.OccurredAt)
	}
	wantMeta := bson.D{{Key: "correlation_id", Value: "corr"}, {Key: "actor_id", Value: "actor"}}
	if !reflect.DeepEqual(doc.Meta, wantMeta) {
		t.Errorf("meta = %v, want %v", doc.Meta, wantMeta)
	}

	payload, ok := doc.Payload.DocumentOK()
	if !ok {
		t.Fatalf("payload is a BSON %s, want a document", doc.Payload.Type)
	}
	if got := payload.Lookup("email").StringValue(); got != "a@example.com" {
		t.Errorf("payload.email = %q", got)
	}
	if got := payload.Lookup("address", "city").StringValue(); got != "Kraków" {
		t.Errorf("payload.address.city = %q", got)
	}
	if got := payload.Lookup("age"); got.Type != bson.TypeInt32 || got.Int32() != 42 {
		t.Errorf("payload.age = %s, want int32 42", got)
	}
	if got := payload.Lookup("tags"); got.Type != bson.TypeArray {
		t.Errorf("payload.tags is a BSON %s, want an array", got.Type)
	}
}

func assertUUID(t *testing.T, field string, v bson.RawValue, want uuid.UUID) {
	t.Helper()
	subtype, data, ok := v.BinaryOK()
	if !ok || subtype != 0x04 || string(data) != string(want[:]) {
		t.Errorf("%s = %s, want %s as binary subtype 4", field, v, want)
	}
}

func TestPayloadKeysMongoTreatsSpecially(t *testing.T) {
	db := mongotest.New(t)
	stream := uuid.New()
	payload := `{"$set":{"a.b":1},"$date":"2026-10-24","nested":{"$numberLong":"7","x.y":[{"$oid":"1"}]},"big":18446744073709551615}`
	if err := appendRecords(t, db, stream, 0, record(stream, 1, payload)); err != nil {
		t.Fatal(err)
	}
	got, err := mongodb.NewEventStore(db).Load(t.Context(), stream)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("loaded %d records, want 1", len(got))
	}
	if string(got[0].Payload) != payload {
		t.Fatalf("loaded payload %s, want %s", got[0].Payload, payload)
	}
}

func TestAppendRejectsPayloadThatIsNotAnObject(t *testing.T) {
	db := mongotest.New(t)
	stream := uuid.New()
	good, bad := record(stream, 1, `{}`), record(stream, 2, `["not","an","object"]`)

	err := appendRecords(t, db, stream, 0, good, bad)
	if err == nil || errors.Is(err, es.ErrConcurrency) || !strings.Contains(err.Error(), "version 2") {
		t.Fatalf("append: err = %v, want a payload error naming version 2", err)
	}
	if got, err := mongodb.NewEventStore(db).Load(t.Context(), stream); err != nil || len(got) != 0 {
		t.Fatalf("load = %d records, %v; want none", len(got), err)
	}
}

func TestAppendNothingStillChecksVersion(t *testing.T) {
	db := mongotest.New(t)
	stream := uuid.New()
	if err := appendRecords(t, db, stream, 0); err != nil {
		t.Errorf("appending nothing to a new stream: %v", err)
	}
	if err := appendRecords(t, db, stream, 1); !errors.Is(err, es.ErrConcurrency) {
		t.Errorf("appending nothing at version 1 of a new stream: err = %v, want es.ErrConcurrency", err)
	}
}

// Regression for C10: the old store sorted by occurred_at, at millisecond
// precision, so events in the same millisecond or with a skewed clock were
// replayed out of order.
func TestLoadSortsByVersionNotTime(t *testing.T) {
	db := mongotest.New(t)
	stream := uuid.New()
	ids := []uuid.UUID{uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}
	insert := func(version int, at time.Time) {
		t.Helper()
		_, err := db.Database.Collection("events").InsertOne(t.Context(), bson.D{
			{Key: "_id", Value: bson.Binary{Subtype: 0x04, Data: ids[version-1][:]}},
			{Key: "stream_id", Value: bson.Binary{Subtype: 0x04, Data: stream[:]}},
			{Key: "stream_type", Value: "user"},
			{Key: "version", Value: version},
			{Key: "type", Value: "user.registered.v1"},
			{Key: "occurred_at", Value: bson.NewDateTimeFromTime(at)},
			{Key: "payload", Value: bson.D{{Key: "v", Value: version}}},
			{Key: "meta", Value: bson.D{}},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	// Inserted out of order, with times running backwards.
	insert(3, t0)
	insert(1, t0.Add(2*time.Second))
	insert(2, t0.Add(time.Second))

	got, err := mongodb.NewEventStore(db).Load(t.Context(), stream)
	if err != nil {
		t.Fatal(err)
	}
	var versions []int
	for _, rec := range got {
		versions = append(versions, rec.Version)
		var p struct{ V int }
		if err := json.Unmarshal(rec.Payload, &p); err != nil || p.V != rec.Version {
			t.Errorf("version %d has payload %s", rec.Version, rec.Payload)
		}
		if rec.ID != ids[rec.Version-1] {
			t.Errorf("version %d has ID %s, want %s", rec.Version, rec.ID, ids[rec.Version-1])
		}
	}
	if !reflect.DeepEqual(versions, []int{1, 2, 3}) {
		t.Fatalf("loaded versions %v, want [1 2 3]", versions)
	}
}

func TestLoadRejectsMalformedDocuments(t *testing.T) {
	tests := []struct {
		name    string
		change  bson.E
		wantErr string
	}{
		{"_id not a UUID", bson.E{Key: "_id", Value: bson.Binary{Subtype: 0x00, Data: make([]byte, 16)}}, "_id: want a UUID"},
		{"_id a string", bson.E{Key: "_id", Value: "an id"}, "_id"},
		{"payload a string", bson.E{Key: "payload", Value: `{"a":1}`}, "payload"},
		{"payload holds a date", bson.E{Key: "payload", Value: bson.D{{Key: "at", Value: bson.NewDateTimeFromTime(t0)}}}, `payload: "at"`},
		{"version a string", bson.E{Key: "version", Value: "1"}, "version"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := mongotest.New(t)
			stream := uuid.New()
			id := uuid.NewV7()
			doc := bson.D{
				{Key: "_id", Value: bson.Binary{Subtype: 0x04, Data: id[:]}},
				{Key: "stream_id", Value: bson.Binary{Subtype: 0x04, Data: stream[:]}},
				{Key: "stream_type", Value: "user"},
				{Key: "version", Value: 1},
				{Key: "type", Value: "user.registered.v1"},
				{Key: "occurred_at", Value: bson.NewDateTimeFromTime(t0)},
				{Key: "payload", Value: bson.D{}},
				{Key: "meta", Value: bson.D{}},
			}
			for i := range doc {
				if doc[i].Key == tt.change.Key {
					doc[i] = tt.change
				}
			}
			if _, err := db.Database.Collection("events").InsertOne(t.Context(), doc); err != nil {
				t.Fatal(err)
			}

			got, err := mongodb.NewEventStore(db).Load(t.Context(), stream)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("load = %d records, err %v; want an error containing %q", len(got), err, tt.wantErr)
			}
		})
	}
}
