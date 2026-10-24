// Package mongodb keeps the user service's stores in MongoDB, in the
// database and transactions of the platform's event store and outbox.
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

	platform "github.com/vardius/go-api-boilerplate/internal/platform/mongodb"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/es"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
	"github.com/vardius/go-api-boilerplate/internal/user/domain"
	"github.com/vardius/go-api-boilerplate/internal/user/store"
)

const (
	usersCollection      = "users"
	loginCodesCollection = "login_codes"
)

// Indexes are the indexes the user stores need. Ensure them, with
// platform.Indexes, before serving.
var Indexes = []platform.Index{
	{
		Collection: usersCollection,
		Model: mongo.IndexModel{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetName("users_email").SetUnique(true),
		},
	},
	{
		// Only a string is a linked account; unlinked users hold null,
		// which the partial filter leaves out of the index, so any number
		// of them fit.
		Collection: usersCollection,
		Model: mongo.IndexModel{
			Keys: bson.D{{Key: "google_sub", Value: 1}},
			Options: options.Index().SetName("users_google_sub").SetUnique(true).
				SetPartialFilterExpression(bson.D{{Key: "google_sub", Value: bson.D{{Key: "$type", Value: "string"}}}}),
		},
	},
	{
		// MongoDB removes expired codes itself, within a minute or so of
		// expires_at; Find and Consume check the time regardless.
		Collection: loginCodesCollection,
		Model: mongo.IndexModel{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetName("login_codes_expires_at").SetExpireAfterSeconds(0),
		},
	},
}

// EnsureIndexes creates the platform's indexes and Indexes.
func EnsureIndexes(ctx context.Context, db *platform.DB) error {
	return platform.EnsureIndexes(ctx, db.Database, append(append([]platform.Index{}, platform.Indexes...), Indexes...)...)
}

// Users is the users projection in the users collection.
type Users struct {
	users *mongo.Collection
}

var _ store.Users = (*Users)(nil)

// NewUsers returns the projection on db, which needs Indexes.
func NewUsers(db *platform.DB) *Users {
	return &Users{users: db.Database.Collection(usersCollection)}
}

// userDoc is a user as the collection holds it. Registration writes
// google_sub as null rather than leaving it out, so every document has the
// same fields, and the partial index skips the null.
type userDoc struct {
	ID        bson.Binary   `bson:"_id"`
	Email     string        `bson:"email"`
	GoogleSub *string       `bson:"google_sub"`
	CreatedAt bson.DateTime `bson:"created_at"`
}

// Project implements store.Users.
func (s *Users) Project(ctx context.Context, tx uow.Tx, events []es.Event) error {
	ctx, err := platform.SessionContext(ctx, tx)
	if err != nil {
		return err
	}
	for _, e := range events {
		if err := s.project(ctx, e); err != nil {
			return fmt.Errorf("mongodb: project %s on user %s: %w", e.Type, e.StreamID, err)
		}
	}
	return nil
}

func (s *Users) project(ctx context.Context, e es.Event) error {
	id := bson.D{{Key: "_id", Value: binaryUUID(e.StreamID)}}
	switch p := e.Payload.(type) {
	case domain.Registered:
		// $setOnInsert leaves a user already projected as it is; the upsert
		// takes _id from the filter.
		_, err := s.users.UpdateOne(ctx, id, bson.D{{Key: "$setOnInsert", Value: bson.D{
			{Key: "email", Value: p.Email.String()},
			{Key: "google_sub", Value: nil},
			{Key: "created_at", Value: bson.NewDateTimeFromTime(e.OccurredAt)},
		}}}, options.UpdateOne().SetUpsert(true))
		if platform.IsDuplicate(err) {
			return store.ErrEmailTaken
		}
		return err
	case domain.EmailChanged:
		return s.set(ctx, id, "email", p.New.String(), store.ErrEmailTaken)
	case domain.GoogleLinked:
		return s.set(ctx, id, "google_sub", p.Subject, store.ErrGoogleTaken)
	default:
		return fmt.Errorf("unknown event %T", e.Payload)
	}
}

func (s *Users) set(ctx context.Context, id bson.D, field, value string, taken error) error {
	res, err := s.users.UpdateOne(ctx, id, bson.D{{Key: "$set", Value: bson.D{{Key: field, Value: value}}}})
	if platform.IsDuplicate(err) {
		return taken
	}
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errors.New("the user isn't registered")
	}
	return nil
}

// ByID implements store.Users.
func (s *Users) ByID(ctx context.Context, id uuid.UUID) (store.User, error) {
	return s.find(ctx, bson.D{{Key: "_id", Value: binaryUUID(id)}})
}

// ByEmail implements store.Users.
func (s *Users) ByEmail(ctx context.Context, email domain.Email) (store.User, error) {
	return s.find(ctx, bson.D{{Key: "email", Value: email.String()}})
}

// ByGoogleSubject implements store.Users.
func (s *Users) ByGoogleSubject(ctx context.Context, subject string) (store.User, error) {
	// A string never matches the null of an unlinked user.
	return s.find(ctx, bson.D{{Key: "google_sub", Value: subject}})
}

func (s *Users) find(ctx context.Context, filter bson.D) (store.User, error) {
	var doc userDoc
	err := s.users.FindOne(ctx, filter).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return store.User{}, fmt.Errorf("mongodb: find user: %w", apperr.ErrNotFound)
	}
	if err != nil {
		return store.User{}, fmt.Errorf("mongodb: find user: %w", err)
	}
	id, err := parseBinaryUUID(doc.ID)
	if err != nil {
		return store.User{}, fmt.Errorf("mongodb: find user: _id: %w", err)
	}
	u := store.User{ID: id, Email: doc.Email, CreatedAt: doc.CreatedAt.Time().UTC()}
	if doc.GoogleSub != nil {
		u.GoogleSubject = *doc.GoogleSub
	}
	return u, nil
}

// LoginCodes keeps login codes in the login_codes collection.
type LoginCodes struct {
	codes *mongo.Collection
}

var _ store.LoginCodes = (*LoginCodes)(nil)

// NewLoginCodes returns the code store on db, which needs Indexes.
func NewLoginCodes(db *platform.DB) *LoginCodes {
	return &LoginCodes{codes: db.Database.Collection(loginCodesCollection)}
}

// codeDoc is a login code as the collection holds it. UserID and UsedAt
// are null until set, never missing, for the same reason as userDoc's.
type codeDoc struct {
	Hash      bson.Binary    `bson:"_id"`
	Purpose   string         `bson:"purpose"`
	Email     string         `bson:"email"`
	UserID    *bson.Binary   `bson:"user_id"`
	Next      string         `bson:"next"`
	ExpiresAt bson.DateTime  `bson:"expires_at"`
	UsedAt    *bson.DateTime `bson:"used_at"`
}

// Add implements store.LoginCodes.
func (s *LoginCodes) Add(ctx context.Context, tx uow.Tx, c store.LoginCode) error {
	ctx, err := platform.SessionContext(ctx, tx)
	if err != nil {
		return err
	}
	doc := codeDoc{
		Hash:      binaryHash(c.Hash),
		Purpose:   string(c.Purpose),
		Email:     c.Email,
		Next:      c.Next,
		ExpiresAt: bson.NewDateTimeFromTime(c.ExpiresAt),
	}
	if c.UserID != uuid.Nil() {
		id := binaryUUID(c.UserID)
		doc.UserID = &id
	}
	if _, err := s.codes.InsertOne(ctx, doc); err != nil {
		if platform.IsDuplicate(err) {
			return fmt.Errorf("mongodb: add login code: %w: %w", apperr.ErrConflict, err)
		}
		return fmt.Errorf("mongodb: add login code: %w", err)
	}
	return nil
}

func usable(hash store.CodeHash, purpose store.Purpose, now time.Time) bson.D {
	return bson.D{
		{Key: "_id", Value: binaryHash(hash)},
		{Key: "purpose", Value: string(purpose)},
		{Key: "used_at", Value: nil},
		{Key: "expires_at", Value: bson.D{{Key: "$gt", Value: bson.NewDateTimeFromTime(now)}}},
	}
}

// Find implements store.LoginCodes.
func (s *LoginCodes) Find(ctx context.Context, hash store.CodeHash, purpose store.Purpose, now time.Time) (store.LoginCode, error) {
	c, err := decodeCode(s.codes.FindOne(ctx, usable(hash, purpose, now)))
	if err != nil {
		return store.LoginCode{}, fmt.Errorf("mongodb: find login code: %w", err)
	}
	return c, nil
}

// Consume implements store.LoginCodes. findOneAndUpdate matches and marks
// the code in one step; of two transactions doing it at once, one hits a
// write conflict, and when the driver retries it the code no longer
// matches.
func (s *LoginCodes) Consume(ctx context.Context, tx uow.Tx, hash store.CodeHash, purpose store.Purpose, now time.Time) (store.LoginCode, error) {
	ctx, err := platform.SessionContext(ctx, tx)
	if err != nil {
		return store.LoginCode{}, err
	}
	use := bson.D{{Key: "$set", Value: bson.D{{Key: "used_at", Value: bson.NewDateTimeFromTime(now)}}}}
	c, err := decodeCode(s.codes.FindOneAndUpdate(ctx, usable(hash, purpose, now), use))
	if err != nil {
		return store.LoginCode{}, fmt.Errorf("mongodb: consume login code: %w", err)
	}
	return c, nil
}

func decodeCode(res *mongo.SingleResult) (store.LoginCode, error) {
	var doc codeDoc
	err := res.Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return store.LoginCode{}, store.ErrCodeUnusable
	}
	if err != nil {
		return store.LoginCode{}, err
	}
	c := store.LoginCode{
		Purpose:   store.Purpose(doc.Purpose),
		Email:     doc.Email,
		Next:      doc.Next,
		ExpiresAt: doc.ExpiresAt.Time().UTC(),
	}
	if len(doc.Hash.Data) != len(c.Hash) {
		return store.LoginCode{}, fmt.Errorf("_id has %d bytes, want %d", len(doc.Hash.Data), len(c.Hash))
	}
	copy(c.Hash[:], doc.Hash.Data)
	if doc.UserID != nil {
		if c.UserID, err = parseBinaryUUID(*doc.UserID); err != nil {
			return store.LoginCode{}, fmt.Errorf("user_id: %w", err)
		}
	}
	return c, nil
}

// DeleteExpired implements store.LoginCodes. The TTL index does this too,
// on its own schedule.
func (s *LoginCodes) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	res, err := s.codes.DeleteMany(ctx, bson.D{{Key: "expires_at", Value: bson.D{{Key: "$lte", Value: bson.NewDateTimeFromTime(now)}}}})
	if err != nil {
		return 0, fmt.Errorf("mongodb: delete expired login codes: %w", err)
	}
	return res.DeletedCount, nil
}

// uuid.UUID is a [16]byte, which the driver would store as an array of
// numbers, so documents hold UUIDs as binary subtype 4, as the platform's
// do.
func binaryUUID(id uuid.UUID) bson.Binary {
	return bson.Binary{Subtype: bson.TypeBinaryUUID, Data: id[:]}
}

func parseBinaryUUID(b bson.Binary) (uuid.UUID, error) {
	if b.Subtype != bson.TypeBinaryUUID || len(b.Data) != len(uuid.UUID{}) {
		return uuid.UUID{}, fmt.Errorf("want a UUID, got %d bytes of binary subtype %#x", len(b.Data), b.Subtype)
	}
	return uuid.UUID(b.Data), nil
}

func binaryHash(h store.CodeHash) bson.Binary {
	return bson.Binary{Subtype: bson.TypeBinaryGeneric, Data: h[:]}
}
