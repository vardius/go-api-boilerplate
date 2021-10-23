package mongo

import (
	"context"
	"errors"
	"fmt"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewUserRepository returns mongo view model repository for user
func NewUserRepository(ctx context.Context, mongoDB *mongo.Database) (persistence.UserRepository, error) {
	collection := mongoDB.Collection("users")

	if _, err := collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "email_address", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "facebook_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "google_id", Value: 1}}, Options: options.Index().SetUnique(true)},
	}); err != nil {
		return nil, apperrors.Wrap(err)
	}

	return &userRepository{
		collection: collection,
	}, nil
}

type userRepository struct {
	collection *mongo.Collection
}

func (r *userRepository) FindAll(ctx context.Context, limit, offset int64) ([]persistence.User, error) {
	findOptions := options.Find().SetLimit(limit).SetSkip(offset)
	filter := bson.M{}

	cur, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	defer cur.Close(ctx)

	var result []persistence.User
	for cur.Next(ctx) {
		var item User
		if err := cur.Decode(&item); err != nil {
			return nil, apperrors.Wrap(err)
		}

		result = append(result, &item)
	}

	return result, nil
}

func (r *userRepository) Get(ctx context.Context, id string) (persistence.User, error) {
	filter := bson.M{
		"user_id": id,
	}

	var result User
	if err := r.collection.FindOne(ctx, filter).Decode(&result); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.Wrap(fmt.Errorf("%s: %w", err, apperrors.ErrNotFound))
		}
		return nil, apperrors.Wrap(err)
	}

	return &result, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (persistence.User, error) {
	filter := bson.M{
		"email_address": email,
	}

	var result User
	if err := r.collection.FindOne(ctx, filter).Decode(&result); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.Wrap(fmt.Errorf("%s: %w", err, apperrors.ErrNotFound))
		}
		return nil, apperrors.Wrap(err)
	}

	return &result, nil
}

func (r *userRepository) GetByFacebookID(ctx context.Context, facebookID string) (persistence.User, error) {
	filter := bson.M{
		"facebook_id": facebookID,
	}

	var result User
	if err := r.collection.FindOne(ctx, filter).Decode(&result); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.Wrap(fmt.Errorf("%s: %w", err, apperrors.ErrNotFound))
		}
		return nil, apperrors.Wrap(err)
	}

	return &result, nil
}

func (r *userRepository) GetByGoogleID(ctx context.Context, googleID string) (persistence.User, error) {
	filter := bson.M{
		"google_id": googleID,
	}

	var result User
	if err := r.collection.FindOne(ctx, filter).Decode(&result); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.Wrap(fmt.Errorf("%s: %w", err, apperrors.ErrNotFound))
		}
		return nil, apperrors.Wrap(err)
	}

	return &result, nil
}

func (r *userRepository) Add(ctx context.Context, u persistence.User) error {
	token := User{
		ID:         u.GetID(),
		Email:      u.GetEmail(),
		FacebookID: u.GetFacebookID(),
		GoogleID:   u.GetGoogleID(),
	}

	if _, err := r.collection.InsertOne(ctx, token); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

func (r *userRepository) UpdateEmail(ctx context.Context, id, email string) error {
	filter := bson.M{
		"user_id": id,
	}
	update := bson.M{
		"$set": bson.M{
			"email_address": email,
		},
	}
	opt := options.FindOneAndUpdate().SetUpsert(false)

	if err := r.collection.FindOneAndUpdate(ctx, filter, update, opt).Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperrors.Wrap(fmt.Errorf("%s: %w", err, apperrors.ErrNotFound))
		}
		return apperrors.Wrap(err)
	}

	return nil
}

func (r *userRepository) UpdateFacebookID(ctx context.Context, id, facebookID string) error {
	filter := bson.M{
		"user_id": id,
	}
	update := bson.M{
		"$set": bson.M{
			"facebook_id": facebookID,
		},
	}
	opt := options.FindOneAndUpdate().SetUpsert(false)

	if err := r.collection.FindOneAndUpdate(ctx, filter, update, opt).Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperrors.Wrap(fmt.Errorf("%s: %w", err, apperrors.ErrNotFound))
		}
		return apperrors.Wrap(err)
	}

	return nil
}

func (r *userRepository) UpdateGoogleID(ctx context.Context, id, googleID string) error {
	filter := bson.M{
		"user_id": id,
	}
	update := bson.M{
		"$set": bson.M{
			"google_id": googleID,
		},
	}
	opt := options.FindOneAndUpdate().SetUpsert(false)

	if err := r.collection.FindOneAndUpdate(ctx, filter, update, opt).Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperrors.Wrap(fmt.Errorf("%s: %w", err, apperrors.ErrNotFound))
		}
		return apperrors.Wrap(err)
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	filter := bson.M{
		"user_id": id,
	}

	if _, err := r.collection.DeleteOne(ctx, filter); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

func (r *userRepository) Count(ctx context.Context) (int64, error) {
	total, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, apperrors.Wrap(err)
	}

	return total, nil
}
