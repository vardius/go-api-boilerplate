package mysql_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"testing/fstest"
	"time"
	"unicode/utf8"
	"uuid"

	mysqldriver "github.com/go-sql-driver/mysql"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/es"
	"github.com/vardius/go-api-boilerplate/internal/platform/es/estest"
	"github.com/vardius/go-api-boilerplate/internal/platform/memdb"
	"github.com/vardius/go-api-boilerplate/internal/platform/mysql"
	"github.com/vardius/go-api-boilerplate/internal/platform/mysql/mysqltest"
	"github.com/vardius/go-api-boilerplate/internal/platform/outbox"
	"github.com/vardius/go-api-boilerplate/internal/platform/outbox/outboxtest"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

func TestEventStore(t *testing.T) {
	estest.TestStore(t, func(t *testing.T) (es.Store, uow.UnitOfWork) {
		db := mysqltest.New(t)
		return mysql.NewEventStore(db), db
	})
}

func TestOutbox(t *testing.T) {
	outboxtest.TestStore(t, func(t *testing.T) (outbox.Store, uow.UnitOfWork) {
		db := mysqltest.New(t)
		return mysql.NewOutbox(db), db
	})
}

var t0 = time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)

func newMessage(t *testing.T) outbox.Message {
	t.Helper()
	m, err := outbox.NewMessage("mail.login_link", map[string]string{"to": "a@example.com"}, t0)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func add(t *testing.T, db *mysql.DB, store *mysql.Outbox, msgs ...outbox.Message) {
	t.Helper()
	if err := db.Do(t.Context(), func(tx uow.Tx) error { return store.Add(t.Context(), tx, msgs...) }); err != nil {
		t.Fatalf("add: %v", err)
	}
}

func claimed(t *testing.T, store *mysql.Outbox, at time.Time) []outbox.Message {
	t.Helper()
	msgs, err := store.Claim(t.Context(), at, time.Minute, 10)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	return msgs
}

func TestOpenReportsAnUnreachableServer(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	const password = "s3cret-p@ss/w:rd"
	db, err := mysql.Open(t.Context(), mysql.Config{Addr: addr, User: "app", Password: password, Database: "users"})
	if err == nil {
		_ = db.Close()
		t.Fatal("Open succeeded with no server listening")
	}
	if !strings.Contains(err.Error(), addr) {
		t.Errorf("error %q doesn't name the address %s", err, addr)
	}
	if strings.Contains(err.Error(), password) {
		t.Errorf("error %q reveals the password", err)
	}
}

func TestIsDuplicate(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"other error", errors.New("Duplicate entry"), false},
		{"duplicate entry", &mysqldriver.MySQLError{Number: 1062}, true},
		{"wrapped duplicate entry", fmt.Errorf("insert: %w", &mysqldriver.MySQLError{Number: 1062}), true},
		{"deadlock", &mysqldriver.MySQLError{Number: 1213}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mysql.IsDuplicate(tt.err); got != tt.want {
				t.Errorf("IsDuplicate(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

type otherTx struct{}

func (otherTx) Backend() string { return "mongodb" }

func TestStoresRejectAnotherBackendsTx(t *testing.T) {
	if _, err := mysql.SQLTx(otherTx{}); err == nil {
		t.Error("SQLTx accepted another backend's transaction")
	}
	if _, err := mysql.SQLTx(nil); err == nil {
		t.Error("SQLTx accepted a nil transaction")
	}

	db := &mysql.DB{} // Never reached: the stores check the Tx first.
	ctx := t.Context()
	stream := uuid.New()
	if err := mysql.NewEventStore(db).Append(ctx, otherTx{}, stream, 0, nil); err == nil {
		t.Error("Append accepted another backend's transaction")
	}
	if err := mysql.NewOutbox(db).Add(ctx, otherTx{}, newMessage(t)); err == nil {
		t.Error("Add accepted another backend's transaction")
	}
	mem := memdb.New()
	err := mem.Do(ctx, func(tx uow.Tx) error { return mysql.NewEventStore(db).Append(ctx, tx, stream, 0, nil) })
	if err == nil {
		t.Error("Append accepted a memdb transaction")
	}
}

// closedDB returns a DB whose pool is closed, so every query fails at once.
func closedDB(t *testing.T) *mysql.DB {
	t.Helper()
	pool, err := sql.Open("mysql", "app:pw@tcp(127.0.0.1:1)/users")
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
	return &mysql.DB{DB: pool}
}

func TestOperationsReportDatabaseErrors(t *testing.T) {
	db := closedDB(t)
	events, box := mysql.NewEventStore(db), mysql.NewOutbox(db)
	id := uuid.New()
	ops := map[string]func(ctx context.Context) error{
		"Do":    func(ctx context.Context) error { return db.Do(ctx, func(uow.Tx) error { return nil }) },
		"Load":  func(ctx context.Context) error { _, err := events.Load(ctx, id); return err },
		"Claim": func(ctx context.Context) error { _, err := box.Claim(ctx, t0, time.Minute, 1); return err },
		"Ack":   func(ctx context.Context) error { return box.Ack(ctx, id) },
		"Retry": func(ctx context.Context) error { return box.Retry(ctx, id, t0, "reason") },
		"Bury":  func(ctx context.Context) error { return box.Bury(ctx, id, "reason") },
	}
	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			if err := op(t.Context()); err == nil {
				t.Errorf("%s on a closed pool succeeded", name)
			}
		})
	}
}

func TestDoRollsBackOnPanic(t *testing.T) {
	db := mysqltest.New(t)
	store := mysql.NewOutbox(db)

	recovered := func() (v any) {
		defer func() { v = recover() }()
		_ = db.Do(t.Context(), func(tx uow.Tx) error {
			if err := store.Add(t.Context(), tx, newMessage(t)); err != nil {
				return err
			}
			panic("boom")
		})
		return nil
	}()

	if recovered != "boom" {
		t.Fatalf("recovered %v from Do, want the function's panic", recovered)
	}
	if inUse := db.Stats().InUse; inUse != 0 {
		t.Errorf("%d connections still held after the panic, want the transaction's returned", inUse)
	}
	if got := claimed(t, store, t0); len(got) != 0 {
		t.Errorf("a panicking transaction kept %d messages", len(got))
	}
}

// widgets is a service's own migrations, numbered from 1 like the platform's.
var widgets = fstest.MapFS{
	"00001_widgets.sql": {Data: []byte("-- +goose Up\nCREATE TABLE widgets (id BINARY(16) PRIMARY KEY, name VARCHAR(64) NOT NULL);\n")},
}

func TestMigrateKeepsEachSetApart(t *testing.T) {
	db := mysqltest.New(t) // The platform migrations are applied.
	ctx := t.Context()

	for range 2 {
		if err := mysql.Migrate(ctx, db.DB, mysql.Migrations, mysql.MigrationsTable); err != nil {
			t.Fatalf("migrating the platform again: %v", err)
		}
		if err := mysql.Migrate(ctx, db.DB, widgets, "goose_widgets"); err != nil {
			t.Fatalf("migrating the service: %v", err)
		}
	}

	for table, want := range map[string]int{mysql.MigrationsTable: 2, "goose_widgets": 1} {
		var version, applied int
		err := db.QueryRowContext(ctx, "SELECT MAX(version_id), COUNT(*) FROM "+table+" WHERE version_id > 0").Scan(&version, &applied)
		if err != nil {
			t.Fatalf("read %s: %v", table, err)
		}
		if version != want || applied != want {
			t.Errorf("%s is at version %d with %d migrations recorded, want %d and %d", table, version, applied, want, want)
		}
	}
	id := uuid.New()
	if _, err := db.ExecContext(ctx, "INSERT INTO widgets (id, name) VALUES (?, 'gear')", id[:]); err != nil {
		t.Errorf("the service's table wasn't created: %v", err)
	}
}

func TestMigrateReportsFailures(t *testing.T) {
	db := mysqltest.New(t)
	tests := map[string]fstest.MapFS{
		"no migrations": {},
		"bad SQL":       {"00001_bad.sql": {Data: []byte("-- +goose Up\nCREATE TABLE;\n")}},
	}
	for name, fsys := range tests {
		t.Run(name, func(t *testing.T) {
			if err := mysql.Migrate(t.Context(), db.DB, fsys, "goose_broken"); err == nil {
				t.Error("Migrate succeeded")
			}
		})
	}
}

func TestDoCommitsServiceWritesWithTheStores(t *testing.T) {
	db := mysqltest.New(t)
	if err := mysql.Migrate(t.Context(), db.DB, widgets, "goose_widgets"); err != nil {
		t.Fatal(err)
	}
	store := mysql.NewOutbox(db)
	addWidget := func(ctx context.Context, tx uow.Tx, m outbox.Message) error {
		sqlTx, err := mysql.SQLTx(tx)
		if err != nil {
			return err
		}
		if _, err := sqlTx.ExecContext(ctx, "INSERT INTO widgets (id, name) VALUES (?, 'gear')", m.ID[:]); err != nil {
			return err
		}
		return store.Add(ctx, tx, m)
	}
	countWidgets := func() int {
		var n int
		if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM widgets").Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	failure := errors.New("a later step failed")
	err := db.Do(t.Context(), func(tx uow.Tx) error {
		if err := addWidget(t.Context(), tx, newMessage(t)); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("Do returned %v, want the function's error", err)
	}
	if n, msgs := countWidgets(), claimed(t, store, t0); n != 0 || len(msgs) != 0 {
		t.Fatalf("a rolled-back transaction kept %d widgets and %d messages", n, len(msgs))
	}

	m := newMessage(t)
	if err := db.Do(t.Context(), func(tx uow.Tx) error { return addWidget(t.Context(), tx, m) }); err != nil {
		t.Fatal(err)
	}
	if n, msgs := countWidgets(), claimed(t, store, t0); n != 1 || len(msgs) != 1 || msgs[0].ID != m.ID {
		t.Fatalf("a committed transaction kept %d widgets and messages %v, want 1 widget and message %s", n, msgs, m.ID)
	}
}

func record(stream uuid.UUID, version int, payload string) es.Record {
	return es.Record{
		ID:         uuid.NewV7(),
		StreamID:   stream,
		StreamType: "test",
		Version:    version,
		Type:       "test.happened.v1",
		OccurredAt: t0,
		Payload:    []byte(payload),
	}
}

func TestAppendRejectsBadRecords(t *testing.T) {
	db := mysqltest.New(t)
	store := mysql.NewEventStore(db)
	stream := uuid.New()
	tests := map[string][]es.Record{
		"another stream":   {record(uuid.New(), 1, `{}`)},
		"version too far":  {record(stream, 2, `{}`)},
		"version gap":      {record(stream, 1, `{}`), record(stream, 3, `{}`)},
		"payload not JSON": {record(stream, 1, `{"n":`)},
	}
	for name, recs := range tests {
		t.Run(name, func(t *testing.T) {
			err := db.Do(t.Context(), func(tx uow.Tx) error { return store.Append(t.Context(), tx, stream, 0, recs) })
			if err == nil || errors.Is(err, es.ErrConcurrency) {
				t.Errorf("Append: err = %v, want an error that isn't a conflict", err)
			}
		})
	}
	if got, err := store.Load(t.Context(), stream); err != nil || len(got) != 0 {
		t.Errorf("Load = %d records, %v; want none stored", len(got), err)
	}
}

func TestAppendNothingStillChecksTheVersion(t *testing.T) {
	db := mysqltest.New(t)
	store := mysql.NewEventStore(db)
	stream := uuid.New()
	appendAt := func(expected int, recs ...es.Record) error {
		return db.Do(t.Context(), func(tx uow.Tx) error { return store.Append(t.Context(), tx, stream, expected, recs) })
	}

	if err := appendAt(0, record(stream, 1, `{}`)); err != nil {
		t.Fatal(err)
	}
	if err := appendAt(1); err != nil {
		t.Errorf("appending nothing at the stream's version: %v", err)
	}
	if err := appendAt(0); !errors.Is(err, es.ErrConcurrency) {
		t.Errorf("appending nothing at a stale version: err = %v, want es.ErrConcurrency", err)
	}
}

func TestLoadReportsUnreadableMeta(t *testing.T) {
	db := mysqltest.New(t)
	id, stream := uuid.New(), uuid.New()
	_, err := db.ExecContext(t.Context(),
		"INSERT INTO events (id, stream_id, stream_type, version, type, payload, meta, occurred_at) VALUES (?, ?, 'test', 1, 'test.happened.v1', '{}', '[1]', ?)",
		id[:], stream[:], t0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mysql.NewEventStore(db).Load(t.Context(), stream); err == nil {
		t.Error("Load returned a record whose meta isn't an object")
	}
}

func TestOutboxAddRejects(t *testing.T) {
	db := mysqltest.New(t)
	store := mysql.NewOutbox(db)
	m := newMessage(t)
	add(t, db, store, m)

	addOnce := func(msgs ...outbox.Message) error {
		return db.Do(t.Context(), func(tx uow.Tx) error { return store.Add(t.Context(), tx, msgs...) })
	}
	if err := addOnce(m); !errors.Is(err, apperr.ErrConflict) {
		t.Errorf("adding a message twice: err = %v, want apperr.ErrConflict", err)
	}
	bad := newMessage(t)
	bad.Payload = []byte(`{"to":`)
	if err := addOnce(bad); err == nil || errors.Is(err, apperr.ErrConflict) {
		t.Errorf("adding a payload that isn't JSON: err = %v, want the server's error", err)
	}
	if err := addOnce(); err != nil {
		t.Errorf("adding no messages: %v", err)
	}
}

func TestOutboxUnknownMessage(t *testing.T) {
	db := mysqltest.New(t)
	store := mysql.NewOutbox(db)
	id := uuid.New()

	if err := store.Retry(t.Context(), id, t0, "reason"); !errors.Is(err, apperr.ErrNotFound) {
		t.Errorf("Retry: err = %v, want apperr.ErrNotFound", err)
	}
	if err := store.Bury(t.Context(), id, "reason"); !errors.Is(err, apperr.ErrNotFound) {
		t.Errorf("Bury: err = %v, want apperr.ErrNotFound", err)
	}
	if err := store.Ack(t.Context(), id); err != nil {
		t.Errorf("Ack: %v, want nil: acking twice is harmless", err)
	}
}

func TestOutboxRepeatedRetryFindsTheMessage(t *testing.T) {
	db := mysqltest.New(t)
	store := mysql.NewOutbox(db)
	m := newMessage(t)
	add(t, db, store, m)
	claimed(t, store, t0)

	// The second call changes nothing; it must still find the row.
	for i := range 2 {
		if err := store.Retry(t.Context(), m.ID, t0.Add(time.Second), "smtp: connection refused"); err != nil {
			t.Fatalf("retry %d: %v", i+1, err)
		}
	}
}

func TestOutboxKeepsAnyReason(t *testing.T) {
	db := mysqltest.New(t)
	store := mysql.NewOutbox(db)
	m := newMessage(t)
	add(t, db, store, m)

	// A response body far bigger than the column, with bytes that aren't
	// UTF-8, made of three-byte runes so that a cut by bytes splits one.
	reason := "smtp: 554 \xff\xfe " + strings.Repeat("✓", 64<<10)
	lastError := func() string {
		var s string
		if err := db.QueryRowContext(t.Context(), "SELECT last_error FROM outbox WHERE id = ?", m.ID[:]).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}

	if err := store.Retry(t.Context(), m.ID, t0, reason); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if got := lastError(); !utf8.ValidString(got) || !strings.HasPrefix(got, "smtp: 554 ") || len(got) > 64<<10 {
		t.Errorf("stored a last error of %d bytes starting %q, want valid UTF-8 that fits the column", len(got), got[:min(len(got), 20)])
	}
	if err := store.Bury(t.Context(), m.ID, reason); err != nil {
		t.Fatalf("bury: %v", err)
	}
	if got := claimed(t, store, t0.Add(time.Hour)); len(got) != 0 {
		t.Errorf("claimed %d buried messages", len(got))
	}
}

func TestClaimWithNoLimitTakesNothing(t *testing.T) {
	db := mysqltest.New(t)
	store := mysql.NewOutbox(db)
	m := newMessage(t)
	add(t, db, store, m)

	for _, limit := range []int{0, -1} {
		got, err := store.Claim(t.Context(), t0, time.Minute, limit)
		if err != nil || len(got) != 0 {
			t.Errorf("Claim(limit %d) = %d messages, %v; want none", limit, len(got), err)
		}
	}
	if got := claimed(t, store, t0); len(got) != 1 || got[0].Attempts != 1 {
		t.Errorf("after claims with no limit, claimed %v, want the message on its first attempt", got)
	}
}
