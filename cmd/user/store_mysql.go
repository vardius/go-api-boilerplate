//go:build persistence_mysql

package main

import (
	"context"

	"github.com/vardius/go-api-boilerplate/internal/platform/mysql"
	"github.com/vardius/go-api-boilerplate/internal/platform/run"
	usermysql "github.com/vardius/go-api-boilerplate/internal/user/store/mysql"
)

type storeConfig struct {
	MySQL mysql.Config `envPrefix:"MYSQL_"`
}

func openStores(ctx context.Context, cfg storeConfig) (*stores, error) {
	db, err := mysql.Open(ctx, cfg.MySQL)
	if err != nil {
		return nil, err
	}
	return &stores{
		uow:     db,
		events:  mysql.NewEventStore(db),
		outbox:  mysql.NewOutbox(db),
		users:   usermysql.NewUsers(db),
		codes:   usermysql.NewLoginCodes(db),
		checks:  map[string]run.Check{"mysql": db.PingContext},
		migrate: func(ctx context.Context) error { return usermysql.Migrate(ctx, db) },
		close:   db.Close,
	}, nil
}
