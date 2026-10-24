package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	eventsCollection = "events"
	outboxCollection = "outbox"
)

// Index is an index a collection needs.
type Index struct {
	Collection string
	Model      mongo.IndexModel
}

// Indexes are the indexes the event store and the outbox need. Ensure them
// at start, with a service's own.
var Indexes = []Index{
	{
		// Keeps one event per stream version, so of two concurrent appends
		// at the same version only one commits.
		Collection: eventsCollection,
		Model: mongo.IndexModel{
			Keys:    bson.D{{Key: "stream_id", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	},
	{
		// Serves Claim's filter and its sort. _id is in the key because the
		// sort breaks ties on it; without it every claim sorts all due
		// messages in memory.
		Collection: outboxCollection,
		Model: mongo.IndexModel{
			Keys: bson.D{{Key: "buried_at", Value: 1}, {Key: "due_at", Value: 1}, {Key: "_id", Value: 1}},
		},
	},
}

// EnsureIndexes creates indexes and the collections they are on. An index
// that already exists with the same definition is left alone, so it is safe
// to run at every start; one whose name exists with another definition is
// an error.
func EnsureIndexes(ctx context.Context, db *mongo.Database, indexes ...Index) error {
	for _, ix := range indexes {
		if _, err := db.Collection(ix.Collection).Indexes().CreateOne(ctx, ix.Model); err != nil {
			return fmt.Errorf("mongodb: create index on %s: %w", ix.Collection, err)
		}
	}
	return nil
}
