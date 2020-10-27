/*
Package eventstore provides mongo implementation of domain event store
*/
package eventstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	baseeventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore"
)

type eventStore struct {
	collection *mongo.Collection
}

// New creates new mongo event store
func New(collectionName string, mongoDB *mongo.Database) baseeventstore.EventStore {
	if collectionName == "" {
		collectionName = "events"
	}

	return &eventStore{
		collection: mongoDB.Collection(collectionName),
	}
}

func (s *eventStore) Store(ctx context.Context, events []domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	var buffer []mongo.WriteModel
	for _, e := range events {
		upsert := mongo.NewInsertOneModel()
		upsert.SetDocument(bson.M{"$set": e})

		buffer = append(buffer, upsert)
	}

	opts := options.BulkWriteOptions{}
	opts.SetOrdered(true)

	const chunkSize = 500

	for i := 0; i < len(buffer); i += chunkSize {
		end := i + chunkSize

		if end > len(buffer) {
			end = len(buffer)
		}

		if _, err := s.collection.BulkWrite(ctx, buffer[i:end], &opts); err != nil {
			return err
		}
	}

	return nil
}

func (s *eventStore) Get(ctx context.Context, id uuid.UUID) (domain.Event, error) {
	filter := bson.M{
		"event_id": id.String(),
	}

	var result domain.Event
	if err := s.collection.FindOne(ctx, filter).Decode(&result); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.NullEvent, apperrors.Wrap(fmt.Errorf("%s: %w", err, baseeventstore.ErrEventNotFound))
		}

		return domain.NullEvent, apperrors.Wrap(err)
	}

	return result, nil
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
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer cur.Close(ctx)

	var result []domain.Event
	for cur.Next(ctx) {
		var event domain.Event
		if err := cur.Decode(&event); err != nil {
			return nil, fmt.Errorf("failed to decode event: %w", err)
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
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer cur.Close(ctx)

	var result []domain.Event
	for cur.Next(ctx) {
		var event domain.Event
		if err := cur.Decode(&event); err != nil {
			return nil, fmt.Errorf("failed to decode event: %w", err)
		}

		result = append(result, event)
	}

	return result, nil
}
