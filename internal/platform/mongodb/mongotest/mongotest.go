// Package mongotest gives tests a MongoDB database of their own.
//
// The first New in a test binary starts one MongoDB container, a single-node
// replica set so that transactions work, and every New after it shares the
// container. Testcontainers' reaper removes the container when the test
// binary exits.
package mongotest

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/vardius/go-api-boilerplate/internal/platform/config"
	platform "github.com/vardius/go-api-boilerplate/internal/platform/mongodb"
)

const image = "mongo:8.3.13"

var (
	startOnce sync.Once
	uri       string
	errStart  error
)

// New returns a connection to a new, empty database with platform.Indexes
// ensured, which is dropped when the test ends. It skips the test under
// -short, since it needs Docker.
func New(t testing.TB) *platform.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("mongotest: needs Docker; skipped with -short")
	}
	startOnce.Do(func() { uri, errStart = start() })
	if errStart != nil {
		t.Fatal(errStart)
	}

	name := "test_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	db, err := platform.Connect(t.Context(), platform.Config{URI: config.Secret(uri), Database: name})
	if err != nil {
		t.Fatalf("mongotest: %v", err)
	}
	t.Cleanup(func() {
		// t.Context is done by the time cleanups run.
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 30*time.Second)
		defer cancel()
		if err := db.Database.Drop(ctx); err != nil {
			t.Errorf("mongotest: drop database %s: %v", name, err)
		}
		if err := db.Client.Disconnect(ctx); err != nil {
			t.Errorf("mongotest: disconnect: %v", err)
		}
	})
	if err := platform.EnsureIndexes(t.Context(), db.Database, platform.Indexes...); err != nil {
		t.Fatalf("mongotest: %v", err)
	}
	return db
}

// start runs the container and returns a URI for it. It outlives the test
// that first calls New, so it has its own context.
func start() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	container, err := mongodb.Run(ctx, image, mongodb.WithReplicaSet("rs0"))
	if err != nil {
		return "", fmt.Errorf("mongotest: start %s: %w", image, err)
	}
	// The replica set is initiated, but its one member takes a moment to
	// become primary, and until then writes fail.
	primary := wait.ForExec([]string{"mongosh", "--quiet", "--eval", "quit(db.hello().isWritablePrimary ? 0 : 1)"}).
		WithStartupTimeout(time.Minute)
	if err := primary.WaitUntilReady(ctx, container); err != nil {
		return "", fmt.Errorf("mongotest: wait for a primary: %w", err)
	}
	conn, err := container.ConnectionString(ctx)
	if err != nil {
		return "", fmt.Errorf("mongotest: %w", err)
	}
	u, err := url.Parse(conn)
	if err != nil {
		return "", fmt.Errorf("mongotest: %w", err)
	}
	// The replica set names its member by the container's address, which
	// the host can't always reach (Docker Desktop can't), so talk to the
	// node directly.
	q := u.Query()
	q.Set("directConnection", "true")
	u.RawQuery = q.Encode()
	return u.String(), nil
}
