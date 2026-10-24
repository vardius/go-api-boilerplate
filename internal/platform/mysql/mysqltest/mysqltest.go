// Package mysqltest runs tests against a real MySQL server in a Docker
// container. It is for tests only; production code never imports it, so
// binaries don't link testcontainers.
package mysqltest

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"

	"github.com/vardius/go-api-boilerplate/internal/platform/config"
	"github.com/vardius/go-api-boilerplate/internal/platform/mysql"
)

// image is the MySQL 8.4 LTS release the tests run on.
const image = "mysql:8.4.13"

// New returns a connection to a new database with mysql.Migrations
// applied, dropped when the test ends. All tests in a test binary share one
// server, started on first use; Ryuk, testcontainers' reaper, removes it
// once the binary exits. New skips the test under -short, since it needs
// Docker.
func New(t testing.TB) *mysql.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("mysqltest: needs Docker; skipped under -short")
	}
	srv, err := shared()
	if err != nil {
		t.Fatalf("mysqltest: %v", err)
	}

	name := "test_" + strings.ToLower(rand.Text())
	if _, err := srv.admin.ExecContext(t.Context(), "CREATE DATABASE "+name); err != nil {
		t.Fatalf("mysqltest: create database %s: %v", name, err)
	}
	t.Cleanup(func() {
		// t.Context is already done when cleanups run.
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Minute)
		defer cancel()
		if _, err := srv.admin.ExecContext(ctx, "DROP DATABASE "+name); err != nil {
			t.Errorf("mysqltest: drop database %s: %v", name, err)
		}
	})

	cfg := srv.cfg
	cfg.Database = name
	db, err := mysql.Open(t.Context(), cfg)
	if err != nil {
		t.Fatalf("mysqltest: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("mysqltest: close database %s: %v", name, err)
		}
	})

	if err := mysql.Migrate(t.Context(), db.DB, mysql.Migrations, mysql.MigrationsTable); err != nil {
		t.Fatalf("mysqltest: %v", err)
	}
	return db
}

type server struct {
	// cfg reaches the server with no database chosen.
	cfg mysql.Config
	// admin creates and drops the tests' databases.
	admin *mysql.DB
}

// shared starts the server once per test binary. A failed start fails
// every test that needs it, without trying again.
var shared = sync.OnceValues(start)

func start() (_ *server, err error) {
	// The server outlives the test that starts it, so its start can't hang
	// off that test's context.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// The password holds the characters a DSN uses as separators, so every
	// test that connects also proves mysql.Open escapes them.
	password := "p@ss/w:rd?" + rand.Text()
	ctr, err := tcmysql.Run(ctx, image, tcmysql.WithUsername("root"), tcmysql.WithPassword(password))
	defer func() {
		// Run may return a container that started but never became ready.
		if err != nil {
			err = errors.Join(err, testcontainers.TerminateContainer(ctr))
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("start %s: %w", image, err)
	}
	addr, err := ctr.PortEndpoint(ctx, "3306/tcp", "")
	if err != nil {
		return nil, fmt.Errorf("find the port of %s: %w", image, err)
	}

	cfg := mysql.Config{
		Addr:            addr,
		User:            "root",
		Password:        config.Secret(password),
		MaxOpenConns:    20,
		MaxIdleConns:    10,
		ConnMaxLifetime: 30 * time.Minute,
	}
	admin, err := mysql.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &server{cfg: cfg, admin: admin}, nil
}
