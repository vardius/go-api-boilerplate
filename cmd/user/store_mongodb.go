//go:build persistence_mongodb

package main

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/vardius/go-api-boilerplate/internal/platform/mongodb"
	"github.com/vardius/go-api-boilerplate/internal/platform/run"
	usermongodb "github.com/vardius/go-api-boilerplate/internal/user/store/mongodb"
)

type storeConfig struct {
	MongoDB mongodb.Config `envPrefix:"MONGODB_"`
}

// disconnectTimeout bounds closing the connection pool at exit.
const disconnectTimeout = 10 * time.Second

func openStores(ctx context.Context, cfg storeConfig) (*stores, error) {
	db, err := mongodb.Connect(ctx, cfg.MongoDB)
	if err != nil {
		return nil, err
	}
	return &stores{
		uow:    db,
		events: mongodb.NewEventStore(db),
		outbox: mongodb.NewOutbox(db),
		users:  usermongodb.NewUsers(db),
		codes:  usermongodb.NewLoginCodes(db),
		checks: map[string]run.Check{
			"mongodb": func(ctx context.Context) error { return db.Client.Ping(ctx, readpref.Primary()) },
		},
		migrate: func(ctx context.Context) error { return usermongodb.EnsureIndexes(ctx, db) },
		close: func() error {
			ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), disconnectTimeout)
			defer cancel()
			return db.Client.Disconnect(ctx)
		},
	}, nil
}
