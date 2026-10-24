//go:build !persistence_mysql && !persistence_mongodb

package main

import (
	"context"
	"log/slog"

	"github.com/vardius/go-api-boilerplate/internal/auth/store/memory"
	"github.com/vardius/go-api-boilerplate/internal/platform/memdb"
)

// storeConfig is empty: the memory store needs nothing.
type storeConfig struct{}

func openStore(_ context.Context, _ storeConfig, log *slog.Logger) (*backend, error) {
	log.Warn("the memory store keeps nothing once the process stops")
	s := memory.New(memdb.New())
	return &backend{
		store:   s,
		purger:  s,
		migrate: func(context.Context) error { return nil },
		close:   func() {},
	}, nil
}
