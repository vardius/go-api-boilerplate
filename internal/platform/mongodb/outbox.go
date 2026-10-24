package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/outbox"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// Outbox is an outbox.Store in the outbox collection. A message's payload
// is kept as the JSON text it was given, and a claim is a lease: a
// claimed_until time that other claims wait out.
type Outbox struct {
	msgs *mongo.Collection
}

var _ outbox.Store = (*Outbox)(nil)

// NewOutbox returns the outbox on db. It needs Indexes.
func NewOutbox(db *DB) *Outbox {
	return &Outbox{msgs: db.Database.Collection(outboxCollection)}
}

// messageDoc is a message as Add writes it. Claim adds claimed_until, Retry
// last_error, and Bury buried_at and last_error.
type messageDoc struct {
	ID        bson.Binary   `bson:"_id"`
	Topic     string        `bson:"topic"`
	Payload   string        `bson:"payload"`
	CreatedAt bson.DateTime `bson:"created_at"`
	DueAt     bson.DateTime `bson:"due_at"`
	Attempts  int           `bson:"attempts"`
}

// Add implements outbox.Store.
func (o *Outbox) Add(ctx context.Context, tx uow.Tx, msgs ...outbox.Message) error {
	ctx, err := SessionContext(ctx, tx)
	if err != nil {
		return err
	}
	if len(msgs) == 0 {
		return nil
	}
	docs := make([]messageDoc, len(msgs))
	for i, m := range msgs {
		created := bson.NewDateTimeFromTime(m.CreatedAt)
		docs[i] = messageDoc{
			ID:        binaryUUID(m.ID),
			Topic:     m.Topic,
			Payload:   string(m.Payload),
			CreatedAt: created,
			DueAt:     created,
		}
	}
	if _, err := o.msgs.InsertMany(ctx, docs); err != nil {
		if IsDuplicate(err) {
			return fmt.Errorf("mongodb: add outbox messages: %w: %w", apperr.ErrConflict, err)
		}
		return fmt.Errorf("mongodb: add outbox messages: %w", err)
	}
	return nil
}

// Claim implements outbox.Store. It leases one message per
// findOneAndUpdate, so two claims never take the same message, and stops
// at limit or when none is due.
func (o *Outbox) Claim(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]outbox.Message, error) {
	at := bson.NewDateTimeFromTime(now)
	due := bson.D{
		{Key: "buried_at", Value: nil},
		{Key: "due_at", Value: bson.D{{Key: "$lte", Value: at}}},
		// Matches a message never claimed, retried or whose lease ended.
		{Key: "claimed_until", Value: bson.D{{Key: "$not", Value: bson.D{{Key: "$gt", Value: at}}}}},
	}
	take := bson.D{
		{Key: "$set", Value: bson.D{{Key: "claimed_until", Value: bson.NewDateTimeFromTime(now.Add(lease))}}},
		{Key: "$inc", Value: bson.D{{Key: "attempts", Value: 1}}},
	}
	opts := options.FindOneAndUpdate().
		SetSort(bson.D{{Key: "due_at", Value: 1}, {Key: "_id", Value: 1}}).
		SetReturnDocument(options.After)

	var claimed []outbox.Message
	for range limit {
		var doc messageDoc
		err := o.msgs.FindOneAndUpdate(ctx, due, take, opts).Decode(&doc)
		if errors.Is(err, mongo.ErrNoDocuments) {
			break
		}
		if err != nil {
			// The messages already leased are claimed again when their
			// lease ends.
			return nil, fmt.Errorf("mongodb: claim outbox messages: %w", err)
		}
		m, err := doc.message()
		if err != nil {
			return nil, fmt.Errorf("mongodb: claim outbox messages: %w", err)
		}
		claimed = append(claimed, m)
	}
	return claimed, nil
}

func (d messageDoc) message() (outbox.Message, error) {
	id, err := parseBinaryUUID(d.ID)
	if err != nil {
		return outbox.Message{}, fmt.Errorf("_id: %w", err)
	}
	return outbox.Message{
		ID:        id,
		Topic:     d.Topic,
		Payload:   []byte(d.Payload),
		CreatedAt: d.CreatedAt.Time().UTC(),
		Attempts:  d.Attempts,
	}, nil
}

// Ack implements outbox.Store. Acking a message that is gone does nothing.
func (o *Outbox) Ack(ctx context.Context, id uuid.UUID) error {
	if _, err := o.msgs.DeleteOne(ctx, bson.D{{Key: "_id", Value: binaryUUID(id)}}); err != nil {
		return fmt.Errorf("mongodb: ack outbox message %s: %w", id, err)
	}
	return nil
}

// Retry implements outbox.Store.
func (o *Outbox) Retry(ctx context.Context, id uuid.UUID, at time.Time, reason string) error {
	return o.update(ctx, id, bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "due_at", Value: bson.NewDateTimeFromTime(at)},
			{Key: "last_error", Value: reason},
		}},
		{Key: "$unset", Value: bson.D{{Key: "claimed_until", Value: ""}}},
	})
}

// Bury implements outbox.Store. The message keeps the server's time of
// burial in buried_at.
func (o *Outbox) Bury(ctx context.Context, id uuid.UUID, reason string) error {
	return o.update(ctx, id, bson.D{
		{Key: "$set", Value: bson.D{{Key: "last_error", Value: reason}}},
		{Key: "$currentDate", Value: bson.D{{Key: "buried_at", Value: true}}},
	})
}

func (o *Outbox) update(ctx context.Context, id uuid.UUID, update bson.D) error {
	res, err := o.msgs.UpdateOne(ctx, bson.D{{Key: "_id", Value: binaryUUID(id)}}, update)
	if err != nil {
		return fmt.Errorf("mongodb: update outbox message %s: %w", id, err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("mongodb: outbox message %s: %w", id, apperr.ErrNotFound)
	}
	return nil
}
