package mysql_test

import (
	"path/filepath"
	"testing"

	platform "github.com/vardius/go-api-boilerplate/internal/platform/mysql"
	"github.com/vardius/go-api-boilerplate/internal/platform/mysql/mysqltest"
	"github.com/vardius/go-api-boilerplate/internal/user/store/mysql"
	"github.com/vardius/go-api-boilerplate/internal/user/store/storetest"
)

func TestStores(t *testing.T) {
	storetest.Test(t, func(t *testing.T) storetest.Backend {
		db := mysqltest.New(t)
		if err := mysql.Migrate(t.Context(), db); err != nil {
			t.Fatal(err)
		}
		return storetest.Backend{
			UnitOfWork: db,
			Events:     platform.NewEventStore(db),
			Users:      mysql.NewUsers(db),
			LoginCodes: mysql.NewLoginCodes(db),
		}
	})
}

func TestMigrateTwiceChangesNothing(t *testing.T) {
	db := mysqltest.New(t)
	for i := range 2 {
		if err := mysql.Migrate(t.Context(), db); err != nil {
			t.Fatalf("migrate %d: %v", i+1, err)
		}
	}
	var n int
	err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM "+mysql.MigrationsTable+" WHERE version_id > 0").Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	if n != len(files) {
		t.Errorf("%s records %d migrations, want one per file in migrations/, %d", mysql.MigrationsTable, n, len(files))
	}
}
