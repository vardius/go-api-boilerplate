package mysql

import (
	"testing"
	"time"
	"uuid"

	mysqldriver "github.com/go-sql-driver/mysql"

	"github.com/vardius/go-api-boilerplate/internal/platform/config"
)

func TestConfigDSN(t *testing.T) {
	passwords := []string{
		"",
		"plain",
		"p@ss/w:rd",
		"@/:@/:",
		"a?b=c&d#e",
		`quote' double" back\slash`,
		"(tcp)/db?parseTime=false",
		"zażółć ✓",
	}
	for _, password := range passwords {
		t.Run(password, func(t *testing.T) {
			cfg := Config{Addr: "db.internal:3307", User: "app", Password: config.Secret(password), Database: "users"}

			got, err := mysqldriver.ParseDSN(cfg.dsn())
			if err != nil {
				t.Fatalf("the driver can't parse the DSN: %v", err)
			}
			if got.Passwd != password || got.User != "app" || got.Addr != "db.internal:3307" || got.Net != "tcp" || got.DBName != "users" {
				t.Errorf("parsed user %q password %q net %q addr %q database %q, want app %q tcp db.internal:3307 users",
					got.User, got.Passwd, got.Net, got.Addr, got.DBName, password)
			}
			if !got.ParseTime || got.Loc != time.UTC {
				t.Errorf("parseTime = %v, loc = %v; want times parsed in UTC", got.ParseTime, got.Loc)
			}
			if got.Collation != "utf8mb4_0900_ai_ci" {
				t.Errorf("collation = %q, want utf8mb4_0900_ai_ci", got.Collation)
			}
			if !got.ClientFoundRows {
				t.Error("clientFoundRows is off, so an UPDATE that changes nothing would look like a missing row")
			}
		})
	}
}

func TestUUIDColumnScan(t *testing.T) {
	want := uuid.MustParse("01920c4e-7a3b-7c1d-8e2f-3a4b5c6d7e8f")
	var got uuid.UUID
	if err := (uuidColumn{&got}).Scan(want[:]); err != nil || got != want {
		t.Fatalf("Scan(16 bytes) = %v, %s; want nil, %s", err, got, want)
	}

	for _, src := range []any{nil, "01920c4e-7a3b-7c1d-8e2f-3a4b5c6d7e8f", []byte{1, 2, 3}, make([]byte, 17)} {
		if err := (uuidColumn{&got}).Scan(src); err == nil {
			t.Errorf("Scan(%#v) succeeded, want an error", src)
		}
	}
}
