package mysql

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migrations creates the tables of the event store and the outbox. Apply
// it under the version table MigrationsTable.
var Migrations = func() fs.FS {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err) // The directory is embedded above, so this can't happen.
	}
	return sub
}()

// MigrationsTable is the goose version table that records which of
// Migrations are applied.
const MigrationsTable = "goose_platform"

// Migrate applies the goose migrations in fsys that db hasn't run yet.
// table is goose's version table: give each set of migrations its own, so
// that the platform's and each service's are numbered independently.
func Migrate(ctx context.Context, db *sql.DB, fsys fs.FS, table string) error {
	p, err := goose.NewProvider(goose.DialectMySQL, db, fsys,
		goose.WithTableName(table),
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return fmt.Errorf("mysql: migrate %s: %w", table, err)
	}
	// p.Close would close db, which belongs to the caller.
	if _, err := p.Up(ctx); err != nil {
		return fmt.Errorf("mysql: migrate %s: %w", table, err)
	}
	return nil
}
