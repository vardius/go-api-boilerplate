package mongo

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
)

// NewTokenRepository returns mongo view model repository for token
func NewTokenRepository(ctx context.Context, mongoDB *mongo.Database) (persistence.TokenRepository, error) {
	collection := mongoDB.Collection("tokens")

	if _, err := collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "token_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "code", Value: 1}}},
		{Keys: bson.D{{Key: "access", Value: 1}}},
		{Keys: bson.D{{Key: "refresh", Value: 1}}},
		{Keys: bson.D{
			{Key: "client_id", Value: 1},
			{Key: "user_id", Value: 1},
		}},
	}); err != nil {
		return nil, apperrors.Wrap(err)
	}

	return &tokenRepository{
		collection: collection,
	}, nil
}

type tokenRepository struct {
	collection *mongo.Collection
}

func (r *tokenRepository) Get(ctx context.Context, id string) (persistence.Token, error) {
	filter := bson.M{
		"token_id": id,
	}

	var result Token
	if err := r.collection.FindOne(ctx, filter).Decode(&result); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.Wrap(fmt.Errorf("%s: %w", err, apperrors.ErrNotFound))
		}
		return nil, apperrors.Wrap(err)
	}

	return &result, nil
}

func (r *tokenRepository) GetByCode(ctx context.Context, code string) (persistence.Token, error) {
	filter := bson.M{
		"code": code,
	}

	var result Token
	if err := r.collection.FindOne(ctx, filter).Decode(&result); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.Wrap(fmt.Errorf("%s: %w", err, apperrors.ErrNotFound))
		}
		return nil, apperrors.Wrap(err)
	}

	return &result, nil
}

func (r *tokenRepository) GetByAccess(ctx context.Context, access string) (persistence.Token, error) {
	filter := bson.M{
		"access": access,
	}

	var result Token
	if err := r.collection.FindOne(ctx, filter).Decode(&result); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.Wrap(fmt.Errorf("%s: %w", err, apperrors.ErrNotFound))
		}
		return nil, apperrors.Wrap(err)
	}

	return &result, nil
}

func (r *tokenRepository) GetByRefresh(ctx context.Context, refresh string) (persistence.Token, error) {
	filter := bson.M{
		"refresh": refresh,
	}

	var result Token
	if err := r.collection.FindOne(ctx, filter).Decode(&result); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.Wrap(fmt.Errorf("%s: %w", err, apperrors.ErrNotFound))
		}
		return nil, apperrors.Wrap(err)
	}

	return &result, nil
}

func (r *tokenRepository) Add(ctx context.Context, t persistence.Token) error {
	ti, err := t.TokenInfo()
	if err != nil {
		return apperrors.Wrap(err)
	}

	var expiredAt time.Time
	if code := ti.GetCode(); code != "" {
		expiredAt = ti.GetCodeCreateAt().Add(ti.GetCodeExpiresIn())
	} else {
		expiredAt = ti.GetAccessCreateAt().Add(ti.GetAccessExpiresIn())

		if refresh := ti.GetRefresh(); refresh != "" {
			expiredAt = ti.GetRefreshCreateAt().Add(ti.GetRefreshExpiresIn())
		}
	}
	expiredAt = expiredAt.UTC()

	token := Token{
		ID:        t.GetID(),
		ClientID:  ti.GetClientID(),
		UserID:    ti.GetUserID(),
		Code:      ti.GetCode(),
		Access:    ti.GetAccess(),
		Refresh:   ti.GetRefresh(),
		UserAgent: t.GetUserAgent(),
		Data:      t.GetData(),
	}

	if !expiredAt.IsZero() {
		token.ExpiredAt = &expiredAt
	}

	if _, err := r.collection.InsertOne(ctx, token); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

func (r *tokenRepository) Delete(ctx context.Context, id string) error {
	filter := bson.M{
		"token_id": id,
	}

	if _, err := r.collection.DeleteOne(ctx, filter); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

func (r *tokenRepository) FindAllByClientID(ctx context.Context, clientID string, limit, offset int64) ([]persistence.Token, error) {
	findOptions := options.Find().SetLimit(limit).SetSkip(offset)
	filter := bson.M{
		"client_id": clientID,
	}

	cur, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	defer cur.Close(ctx)

	var result []persistence.Token
	for cur.Next(ctx) {
		var item Token
		if err := cur.Decode(&item); err != nil {
			return nil, apperrors.Wrap(err)
		}

		result = append(result, &item)
	}

	return result, nil
}

func (r *tokenRepository) CountByClientID(ctx context.Context, clientID string) (int64, error) {
	filter := bson.M{
		"client_id": clientID,
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, apperrors.Wrap(err)
	}

	return total, nil
}
