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
	// checks are the readiness checks.
	checks map[string]run.Check
	// purger deletes what expired, every purgeInterval; nil where TTL
	// indexes do it.
	purger store.Purger
	// migrate applies the store's migrations, or ensures its indexes.
	migrate func(context.Context) error
	close   func()
}

// purgeInterval is how often the purger deletes expired grants, tokens,
// requests and codes.
const purgeInterval = 10 * time.Minute
