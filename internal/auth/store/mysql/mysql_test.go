package mysql

import (
	"slices"
	"testing"

	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/auth/store/storetest"
	platform "github.com/vardius/go-api-boilerplate/internal/platform/mysql"
	"github.com/vardius/go-api-boilerplate/internal/platform/mysql/mysqltest"
)

func migrated(t *testing.T) *platform.DB {
	t.Helper()
	db := mysqltest.New(t)
	if err := Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestStore(t *testing.T) {
	storetest.Run(t, func(t *testing.T) store.Store { return New(migrated(t)) })
}

func TestMigrateCreatesTheNamedTables(t *testing.T) {
	db := migrated(t)
	if err := Migrate(t.Context(), db); err != nil {
		t.Fatalf("migrating again: %v", err)
	}
	rows, err := db.QueryContext(t.Context(), "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE()")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{MigrationsTable, clientsTable, grantsTable, tokensTable, requestsTable, codesTable} {
		if !slices.Contains(tables, want) {
			t.Errorf("no table %s among %v", want, tables)
		}
	}
}
