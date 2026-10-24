// Package mysql runs the platform's stores on MySQL 8.4: a unit of work
// over database transactions, an event store, an outbox, and goose
// migrations for their tables and for each service's own.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"uuid"

	mysqldriver "github.com/go-sql-driver/mysql"

	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// Config names a MySQL database and sizes the connection pool. Services
// nest it under their own prefix.
type Config struct {
	Addr string `env:"ADDR" envDefault:"localhost:3306" desc:"MySQL server address, host:port."`
	User string `env:"USER,required" desc:"MySQL user name."`
	// TODO(config.Secret): make Password a config.Secret once that type lands.
	Password        string        `env:"PASSWORD,required" desc:"MySQL password."`
	Database        string        `env:"DATABASE,required" desc:"MySQL database (schema) name."`
	MaxOpenConns    int           `env:"MAX_OPEN_CONNS" envDefault:"20" desc:"Most connections open at once."`
	MaxIdleConns    int           `env:"MAX_IDLE_CONNS" envDefault:"10" desc:"Most idle connections kept for reuse."`
	ConnMaxLifetime time.Duration `env:"CONN_MAX_LIFETIME" envDefault:"30m" desc:"Longest a connection is reused before it is replaced."`
}

// dsn builds the driver's connection string. FormatDSN escapes what needs
// escaping, so a password may hold any character.
func (c Config) dsn() string {
	d := mysqldriver.NewConfig() // Loc is UTC: times are written and read in UTC.
	d.Net = "tcp"
	d.Addr = c.Addr
	d.User = c.User
	d.Passwd = c.Password
	d.DBName = c.Database
	d.ParseTime = true
	d.Collation = "utf8mb4_0900_ai_ci"
	d.ClientFoundRows = true
	return d.FormatDSN()
}

// Open connects to the database cfg names, sizes the pool and checks that
// the server answers. Connections use utf8mb4 and read and write times in
// UTC. An UPDATE's RowsAffected counts the rows it matched, not only those
// it changed, so zero means no row matched.
func Open(ctx context.Context, cfg Config) (*DB, error) {
	db, err := sql.Open("mysql", cfg.dsn())
	if err != nil {
		return nil, fmt.Errorf("mysql: open %s: %w", cfg.Addr, err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	if err := db.PingContext(ctx); err != nil {
		// The ping failure is the error to report; closing a pool that
		// never connected has nothing to add.
		_ = db.Close()
		return nil, fmt.Errorf("mysql: ping %s: %w", cfg.Addr, err)
	}
	return &DB{DB: db}, nil
}

// DB is a connection pool. Its Do runs functions in transactions, and the
// stores built on it take the Tx that Do hands out.
type DB struct {
	*sql.DB
}

var _ uow.UnitOfWork = (*DB)(nil)

// Do runs fn in a transaction at the server's default isolation level,
// REPEATABLE READ. It commits when fn returns nil and rolls back when fn
// returns an error or panics; a panic continues once the rollback is done.
func (db *DB) Do(ctx context.Context, fn func(tx uow.Tx) error) error {
	return db.do(ctx, nil, func(tx *sql.Tx) error { return fn(&Tx{tx: tx}) })
}

func (db *DB) do(ctx context.Context, opts *sql.TxOptions, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("mysql: begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			// fn's error, or its panic, is what the caller needs. A rollback
			// that fails leaves nothing behind: the server discards a
			// transaction whose connection is gone.
			_ = tx.Rollback()
		}
	}()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mysql: commit: %w", err)
	}
	committed = true
	return nil
}

// Tx is a MySQL transaction. Stores reach the *sql.Tx behind it with SQLTx.
type Tx struct {
	tx *sql.Tx
}

var _ uow.Tx = (*Tx)(nil)

// Backend implements uow.Tx.
func (*Tx) Backend() string { return "mysql" }

// SQLTx returns the database transaction behind tx, for service stores that
// write in the same transaction as the event store. It fails for another
// backend's Tx.
func SQLTx(tx uow.Tx) (*sql.Tx, error) {
	t, ok := tx.(*Tx)
	if !ok || t == nil {
		return nil, fmt.Errorf("mysql: got a %s transaction", backend(tx))
	}
	return t.tx, nil
}

func backend(tx uow.Tx) string {
	if tx == nil {
		return "nil"
	}
	return tx.Backend()
}

// erDupEntry is MySQL's ER_DUP_ENTRY: a row would repeat a unique key.
const erDupEntry = 1062

// IsDuplicate reports whether err is MySQL's duplicate-key error.
func IsDuplicate(err error) bool {
	e, ok := errors.AsType[*mysqldriver.MySQLError](err)
	return ok && e.Number == erDupEntry
}

// valueGroups returns n copies of group separated by commas, the VALUES
// list of a multi-row INSERT or the list inside IN (…).
func valueGroups(n int, group string) string {
	return strings.Join(slices.Repeat([]string{group}, n), ", ")
}

// uuidColumn scans a BINARY(16) column into a uuid.UUID.
type uuidColumn struct {
	dst *uuid.UUID
}

func (c uuidColumn) Scan(src any) error {
	b, ok := src.([]byte)
	if !ok || len(b) != len(c.dst) {
		return fmt.Errorf("mysql: scan a UUID from %T of %d bytes, want 16 bytes", src, len(b))
	}
	copy(c.dst[:], b)
	return nil
}
