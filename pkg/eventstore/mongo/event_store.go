package eventstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	baseeventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore"
	appmongo "github.com/vardius/go-api-boilerplate/pkg/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type dto struct {
	ID            string                  `bson:"event_id"`
	Type          string                  `bson:"event_type"`
	StreamID      string                  `bson:"stream_id"`
	StreamName    string                  `bson:"stream_name"`
	StreamVersion int                     `bson:"stream_version"`
	OccurredAt    time.Time               `bson:"occurred_at"`
	Payload       appmongo.JSONRawMessage `bson:"payload"`
	Metadata      appmongo.JSONRawMessage `bson:"metadata,omitempty"`
}

func (o *dto) ToEvent() (domain.Event, error) {
	id, err := uuid.Parse(o.ID)
	if err != nil {
		return domain.NullEvent, apperrors.Wrap(fmt.Errorf("failed to parse id:%s: %w", o.ID, err))
	}
	streamID, err := uuid.Parse(o.StreamID)
	if err != nil {
		return domain.NullEvent, apperrors.Wrap(fmt.Errorf("failed to parse strem id:%s: %w", o.StreamID, err))
	}
	return domain.Event{
		ID:            id,
		StreamID:      streamID,
		Type:          o.Type,
		StreamName:    o.StreamName,
		StreamVersion: o.StreamVersion,
		OccurredAt:    o.OccurredAt,
		Payload:       json.RawMessage(o.Payload),
		Metadata:      json.RawMessage(o.Metadata),
	}, nil
}

func dtoFromEvent(e *domain.Event) *dto {
	return &dto{
		ID:            e.ID.String(),
		Type:          e.Type,
		StreamID:      e.StreamID.String(),
		StreamName:    e.StreamName,
		StreamVersion: e.StreamVersion,
		OccurredAt:    e.OccurredAt,
		Payload:       appmongo.JSONRawMessage(e.Payload),
		Metadata:      appmongo.JSONRawMessage(e.Metadata),
	}
}

type eventStore struct {
	collection *mongo.Collection
}

// New creates new mongo event store
func New(ctx context.Context, collectionName string, mongoDB *mongo.Database) (baseeventstore.EventStore, error) {
	if collectionName == "" {
		collectionName = "events"
	}

	collection := mongoDB.Collection(collectionName)

	if _, err := collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "event_id", Value: -1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "stream_id", Value: -1}}},
		{Keys: bson.D{{Key: "occurred_at", Value: 1}}},
		{Keys: bson.D{
			{Key: "stream_id", Value: 1},
			{Key: "stream_name", Value: 1},
			{Key: "event_type", Value: 1},
			{Key: "occurred_at", Value: 1},
		}},
	}); err != nil {
		return nil, apperrors.Wrap(fmt.Errorf("failed to create indexes: %w", err))
	}

	return &eventStore{
		collection: collection,
	}, nil
}

func (s *eventStore) Store(ctx context.Context, events []domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	var buffer []mongo.WriteModel
	for _, e := range events {
		upsert := mongo.NewInsertOneModel()
		upsert.SetDocument(dtoFromEvent(&e))

		buffer = append(buffer, upsert)
	}

	opts := options.BulkWrite()
	opts.SetOrdered(true)

	const chunkSize = 500

	for i := 0; i < len(buffer); i += chunkSize {
		end := i + chunkSize

		if end > len(buffer) {
			end = len(buffer)
		}

		if _, err := s.collection.BulkWrite(ctx, buffer[i:end], opts); err != nil {
			return err
		}
	}

	return nil
}

func (s *eventStore) Get(ctx context.Context, id uuid.UUID) (domain.Event, error) {
	filter := bson.M{
		"event_id": id.String(),
	}

	var result dto
	if err := s.collection.FindOne(ctx, filter).Decode(&result); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.NullEvent, apperrors.Wrap(fmt.Errorf("%s: %w", err, baseeventstore.ErrEventNotFound))
		}

		return domain.NullEvent, apperrors.Wrap(err)
	}

	event, err := result.ToEvent()
	if err != nil {
		return domain.NullEvent, apperrors.Wrap(err)
	}

	return event, nil
}

func (s *eventStore) FindAll(ctx context.Context) ([]domain.Event, error) {
	filter := bson.M{}
	findOptions := options.FindOptions{
		Sort: bson.D{
			primitive.E{Key: "occurred_at", Value: 1},
		},
	}

	cur, err := s.collection.Find(ctx, filter, &findOptions)
	if err != nil {
		return nil, apperrors.Wrap(fmt.Errorf("failed to query events: %w", err))
	}
	defer cur.Close(ctx)

	var result []domain.Event
	for cur.Next(ctx) {
		var o dto
		if err := cur.Decode(&o); err != nil {
			return nil, apperrors.Wrap(fmt.Errorf("failed to decode event: %w", err))
		}
		event, err := o.ToEvent()
		if err != nil {
			return nil, apperrors.Wrap(err)
		}
		result = append(result, event)
	}

	return result, nil
}

func (s *eventStore) GetStream(ctx context.Context, streamID uuid.UUID, streamName string) ([]domain.Event, error) {
	filter := bson.M{
		"stream_id":   streamID.String(),
		"stream_name": streamName,
	}
	findOptions := options.FindOptions{
		Sort: bson.D{
			primitive.E{Key: "occurred_at", Value: 1},
		},
	}

	cur, err := s.collection.Find(ctx, filter, &findOptions)
	if err != nil {
		return nil, apperrors.Wrap(fmt.Errorf("failed to query events: %w", err))
	}
	defer cur.Close(ctx)

	var result []domain.Event
	for cur.Next(ctx) {
		var o dto
		if err := cur.Decode(&o); err != nil {
			return nil, apperrors.Wrap(fmt.Errorf("failed to decode event: %w", err))
		}
		event, err := o.ToEvent()
		if err != nil {
			return nil, apperrors.Wrap(err)
		}
		result = append(result, event)
	}

	return result, nil
}

func (s *eventStore) GetStreamEventsByType(ctx context.Context, streamID uuid.UUID, streamName, eventType string) ([]domain.Event, error) {
	filter := bson.M{
		"stream_id":   streamID.String(),
		"stream_name": streamName,
		"event_type":  eventType,
	}
	findOptions := options.FindOptions{
		Sort: bson.D{
			primitive.E{Key: "occurred_at", Value: 1},
		},
	}

	cur, err := s.collection.Find(ctx, filter, &findOptions)
	if err != nil {
		return nil, apperrors.Wrap(fmt.Errorf("failed to query events: %w", err))
	}
	defer cur.Close(ctx)

	var result []domain.Event
	for cur.Next(ctx) {
		var o dto
		if err := cur.Decode(&o); err != nil {
			return nil, apperrors.Wrap(fmt.Errorf("failed to decode event: %w", err))
		}
		event, err := o.ToEvent()
		if err != nil {
			return nil, apperrors.Wrap(err)
		}
		result = append(result, event)
	}

	return result, nil
}
