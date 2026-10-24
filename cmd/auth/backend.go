package main

import (
	"context"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/run"
)

// backend is the store a build was compiled with, opened: openStore, in
// the one store_*.go file the build tags pick, returns it.
type backend struct {
	store store.Store
	// checks are the readiness checks; workers run beside the server.
	checks  map[string]run.Check
	workers []run.Server
	// migrate applies the store's migrations, or ensures its indexes.
	migrate func(context.Context) error
	close   func()
}

// purgeInterval is how often expired grants, tokens, requests and codes
// are deleted where no TTL index does it.
const purgeInterval = 10 * time.Minute
