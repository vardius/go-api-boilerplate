//go:build persistence_mysql

package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	authmysql "github.com/vardius/go-api-boilerplate/internal/auth/store/mysql"
	"github.com/vardius/go-api-boilerplate/internal/platform/mysql"
	"github.com/vardius/go-api-boilerplate/internal/platform/run"
)

type storeConfig struct {
	MySQL mysql.Config `envPrefix:"MYSQL_"`
}

func openStore(ctx context.Context, cfg storeConfig, log *slog.Logger) (*backend, error) {
	db, err := mysql.Open(ctx, cfg.MySQL)
	if err != nil {
		return nil, err
	}
	s := authmysql.New(db)
	return &backend{
		store:   s,
		checks:  map[string]run.Check{"mysql": db.PingContext},
		workers: []run.Server{store.PurgeWorker(s, purgeInterval, time.Now, log)},
		migrate: func(ctx context.Context) error { return authmysql.Migrate(ctx, db) },
		close:   func() { _ = db.Close() },
	}, nil
}
