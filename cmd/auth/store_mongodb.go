//go:build persistence_mongodb

package main

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	authmongodb "github.com/vardius/go-api-boilerplate/internal/auth/store/mongodb"
	"github.com/vardius/go-api-boilerplate/internal/platform/mongodb"
	"github.com/vardius/go-api-boilerplate/internal/platform/run"
)

type storeConfig struct {
	MongoDB mongodb.Config `envPrefix:"MONGODB_"`
}

// openStore connects to MongoDB, whose TTL indexes purge what expires, so
// no worker runs.
func openStore(ctx context.Context, cfg storeConfig, _ *slog.Logger) (*backend, error) {
	db, err := mongodb.Connect(ctx, cfg.MongoDB)
	if err != nil {
		return nil, err
	}
	return &backend{
		store:   authmongodb.New(db),
		checks:  map[string]run.Check{"mongodb": func(ctx context.Context) error { return db.Client.Ping(ctx, readpref.Primary()) }},
		migrate: func(ctx context.Context) error { return authmongodb.EnsureIndexes(ctx, db) },
		close:   func() { _ = db.Client.Disconnect(context.Background()) },
	}, nil
}
