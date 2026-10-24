package mongodb

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/vardius/go-api-boilerplate/internal/platform/es"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// EventStore is an es.Store in the events collection. Each event is one
// document whose _id is the event's ID, and its payload is stored as a
// document converted from the record's JSON.
type EventStore struct {
	events *mongo.Collection
}

var _ es.Store = (*EventStore)(nil)

// NewEventStore returns the event store on db. It needs Indexes.
func NewEventStore(db *DB) *EventStore {
	return &EventStore{events: db.Database.Collection(eventsCollection)}
}

type eventDoc struct {
	ID         bson.Binary   `bson:"_id"`
	StreamID   bson.Binary   `bson:"stream_id"`
	StreamType string        `bson:"stream_type"`
	Version    int           `bson:"version"`
	Type       string        `bson:"type"`
	OccurredAt bson.DateTime `bson:"occurred_at"`
	Payload    bson.D        `bson:"payload"`
	Meta       es.Meta       `bson:"meta"`
}

// Append implements es.Store. It reads the stream's version in the
// transaction, and the unique index on stream and version turns a
// concurrent append at the same version into a conflict.
func (s *EventStore) Append(ctx context.Context, tx uow.Tx, streamID uuid.UUID, expectedVersion int, records []es.Record) error {
	ctx, err := SessionContext(ctx, tx)
	if err != nil {
		return err
	}
	version, err := s.version(ctx, streamID)
	if err != nil {
		return fmt.Errorf("mongodb: append to stream %s: %w", streamID, err)
	}
	if version != expectedVersion {
		return es.ErrConcurrency
	}
	if len(records) == 0 {
		return nil
	}

	docs := make([]eventDoc, len(records))
	for i, rec := range records {
		payload, err := jsonToBSON(rec.Payload)
		if err != nil {
			return fmt.Errorf("mongodb: append to stream %s: payload of version %d: %w", streamID, rec.Version, err)
		}
		docs[i] = eventDoc{
			ID:         binaryUUID(rec.ID),
			StreamID:   binaryUUID(streamID),
			StreamType: rec.StreamType,
			Version:    rec.Version,
			Type:       rec.Type,
			OccurredAt: bson.NewDateTimeFromTime(rec.OccurredAt),
			Payload:    payload,
			Meta:       rec.Meta,
		}
	}
	if _, err := s.events.InsertMany(ctx, docs); err != nil {
		if IsDuplicate(err) {
			return es.ErrConcurrency
		}
		// A write conflict keeps its label, so Do retries the transaction
		// and the version read above sees the other append.
		return fmt.Errorf("mongodb: append to stream %s: %w", streamID, err)
	}
	return nil
}

// version returns the stream's last version, or 0 for a new stream.
func (s *EventStore) version(ctx context.Context, streamID uuid.UUID) (int, error) {
	var head struct {
		Version int `bson:"version"`
	}
	err := s.events.FindOne(ctx,
		bson.D{{Key: "stream_id", Value: binaryUUID(streamID)}},
		options.FindOne().
			SetSort(bson.D{{Key: "version", Value: -1}}).
			SetProjection(bson.D{{Key: "version", Value: 1}, {Key: "_id", Value: 0}}),
	).Decode(&head)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read version: %w", err)
	}
	return head.Version, nil
}

// Load implements es.Store.
func (s *EventStore) Load(ctx context.Context, streamID uuid.UUID) ([]es.Record, error) {
	cur, err := s.events.Find(ctx,
		bson.D{{Key: "stream_id", Value: binaryUUID(streamID)}},
		options.Find().SetSort(bson.D{{Key: "version", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("mongodb: load stream %s: %w", streamID, err)
	}
	defer func() { _ = cur.Close(ctx) }()

	var records []es.Record
	for cur.Next(ctx) {
		var doc eventDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, fmt.Errorf("mongodb: load stream %s: %w", streamID, err)
		}
		rec, err := doc.record()
		if err != nil {
			return nil, fmt.Errorf("mongodb: load stream %s version %d: %w", streamID, doc.Version, err)
		}
		records = append(records, rec)
	}
	// A cursor that ends early, on a network error or a killed query, ends
	// Next as if the stream were complete.
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("mongodb: load stream %s: %w", streamID, err)
	}
	return records, nil
}

func (d eventDoc) record() (es.Record, error) {
	id, err := parseBinaryUUID(d.ID)
	if err != nil {
		return es.Record{}, fmt.Errorf("_id: %w", err)
	}
	streamID, err := parseBinaryUUID(d.StreamID)
	if err != nil {
		return es.Record{}, fmt.Errorf("stream_id: %w", err)
	}
	payload, err := bsonToJSON(d.Payload)
	if err != nil {
		return es.Record{}, fmt.Errorf("payload: %w", err)
	}
	return es.Record{
		ID:         id,
		StreamID:   streamID,
		StreamType: d.StreamType,
		Version:    d.Version,
		Type:       d.Type,
		OccurredAt: d.OccurredAt.Time().UTC(),
		Payload:    payload,
		Meta:       d.Meta,
	}, nil
}
