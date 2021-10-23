package mongo

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/oauth2.v4"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
)

type clientRepository struct {
	cfg        *config.Config
	collection *mongo.Collection
}

// NewClientRepository returns mongo view model repository for client
func NewClientRepository(ctx context.Context, cfg *config.Config, mongoDB *mongo.Database) (persistence.ClientRepository, error) {
	collection := mongoDB.Collection("clients")

	if _, err := collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "client_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "domain", Value: 1},
		}},
	}); err != nil {
		return nil, apperrors.Wrap(err)
	}

	return &clientRepository{
		cfg:        cfg,
		collection: collection,
	}, nil
}

func (r *clientRepository) GetByID(ctx context.Context, id string) (oauth2.ClientInfo, error) {
	c, err := r.Get(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	return c, nil
}

func (r *clientRepository) Get(ctx context.Context, id string) (persistence.Client, error) {
	filter := bson.M{
		"client_id": id,
	}

	var result Client
	if err := r.collection.FindOne(ctx, filter).Decode(&result); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.Wrap(fmt.Errorf("%s: %w", err, apperrors.ErrNotFound))
		}
		return nil, apperrors.Wrap(err)
	}

	return &result, nil
}

func (r *clientRepository) FindAllByUserID(ctx context.Context, userID string, limit, offset int64) ([]persistence.Client, error) {
	findOptions := options.Find().SetLimit(limit).SetSkip(offset)
	filter := bson.M{
		"user_id": userID,
		"domain":  bson.M{"$ne": r.cfg.App.Domain},
	}

	cur, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	defer cur.Close(ctx)

	var result []persistence.Client
	for cur.Next(ctx) {
		var item Client
		if err := cur.Decode(&item); err != nil {
			return nil, apperrors.Wrap(err)
		}

		result = append(result, &item)
	}

	return result, nil
}

func (r *clientRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	filter := bson.M{
		"user_id": userID,
		"domain":  bson.M{"$ne": r.cfg.App.Domain},
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, apperrors.Wrap(err)
	}

	return total, nil
}

func (r *clientRepository) Add(ctx context.Context, c persistence.Client) error {
	client := Client{
		ID:          c.GetID(),
		UserID:      c.GetUserID(),
		Secret:      c.GetSecret(),
		Domain:      c.GetDomain(),
		RedirectURL: c.GetRedirectURL(),
		Scopes:      c.GetScopes(),
	}

	if _, err := r.collection.InsertOne(ctx, client); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

func (r *clientRepository) Delete(ctx context.Context, id string) error {
	filter := bson.M{
		"client_id": id,
	}

	if _, err := r.collection.DeleteOne(ctx, filter); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}
