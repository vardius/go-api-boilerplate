//go:build !persistence_mysql && !persistence_mongodb

package main

import (
	"context"

	"github.com/vardius/go-api-boilerplate/internal/platform/memdb"
	"github.com/vardius/go-api-boilerplate/internal/user/store/memory"
)

// storeConfig is empty: the memory store keeps everything in the process,
// and loses it when the process stops.
type storeConfig struct{}

func openStores(context.Context, storeConfig) (*stores, error) {
	db := memdb.New()
	return &stores{
		uow:     db,
		events:  memdb.NewEventStore(db),
		outbox:  memdb.NewOutbox(db),
		users:   memory.NewUsers(db),
		codes:   memory.NewLoginCodes(db),
		migrate: func(context.Context) error { return nil },
		close:   func() error { return nil },
	}, nil
}
