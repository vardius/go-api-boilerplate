// Package mysql keeps the user service's stores in MySQL, in the database
// and transactions of the platform's event store and outbox.
package mysql

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"time"
	"uuid"

	platform "github.com/vardius/go-api-boilerplate/internal/platform/mysql"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/es"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
	"github.com/vardius/go-api-boilerplate/internal/user/domain"
	"github.com/vardius/go-api-boilerplate/internal/user/store"
)

// The tables, as migrations/ creates them.
const (
	usersTable      = "users"
	loginCodesTable = "login_codes"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migrations creates the users and login_codes tables. Apply it under the
// version table MigrationsTable, or with Migrate.
var Migrations = func() fs.FS {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err) // The directory is embedded above, so this can't happen.
	}
	return sub
}()

// MigrationsTable is the goose version table that records which of
// Migrations are applied, apart from the platform's.
const MigrationsTable = "goose_user"

// Migrate applies the platform's migrations and the user service's that db
// hasn't run yet.
func Migrate(ctx context.Context, db *platform.DB) error {
	if err := platform.Migrate(ctx, db.DB, platform.Migrations, platform.MigrationsTable); err != nil {
		return err
	}
	return platform.Migrate(ctx, db.DB, Migrations, MigrationsTable)
}

// Users is the users projection in the users table.
type Users struct {
	db *platform.DB
}

var _ store.Users = (*Users)(nil)

// NewUsers returns the projection on db, which needs Migrations.
func NewUsers(db *platform.DB) *Users { return &Users{db: db} }

const (
	qUserExists     = "SELECT 1 FROM " + usersTable + " WHERE id = ?"
	qInsertUser     = "INSERT INTO " + usersTable + " (id, email, created_at) VALUES (?, ?, ?)"
	qSetEmail       = "UPDATE " + usersTable + " SET email = ? WHERE id = ?"
	qSetGoogle      = "UPDATE " + usersTable + " SET google_sub = ? WHERE id = ?"
	userColumns     = "SELECT id, email, google_sub, created_at FROM " + usersTable
	qUserByID       = userColumns + " WHERE id = ?"
	qUserByEmail    = userColumns + " WHERE email = ?"
	qUserBySubject  = userColumns + " WHERE google_sub = ?"
	qInsertCode     = "INSERT INTO " + loginCodesTable + " (code_hash, purpose, email, user_id, `next`, expires_at) VALUES (?, ?, ?, ?, ?, ?)"
	codeColumns     = "SELECT code_hash, purpose, email, user_id, `next`, expires_at FROM " + loginCodesTable
	qUsableCode     = codeColumns + " WHERE code_hash = ? AND purpose = ? AND used_at IS NULL AND expires_at > ?"
	qCodeByHash     = codeColumns + " WHERE code_hash = ?"
	qConsumeCode    = "UPDATE " + loginCodesTable + " SET used_at = ? WHERE code_hash = ? AND purpose = ? AND used_at IS NULL AND expires_at > ?"
	qDeleteExpired  = "DELETE FROM " + loginCodesTable + " WHERE expires_at <= ? LIMIT ?"
	deleteBatchSize = 1000
)

// Project implements store.Users.
func (s *Users) Project(ctx context.Context, tx uow.Tx, events []es.Event) error {
	t, err := platform.SQLTx(tx)
	if err != nil {
		return err
	}
	for _, e := range events {
		if err := project(ctx, t, e); err != nil {
			return fmt.Errorf("mysql: project %s on user %s: %w", e.Type, e.StreamID, err)
		}
	}
	return nil
}

func project(ctx context.Context, t *sql.Tx, e es.Event) error {
	switch p := e.Payload.(type) {
	case domain.Registered:
		// An insert that hits the email key must fail, so idempotence comes
		// from looking first rather than from ON DUPLICATE KEY UPDATE.
		var one int
		err := t.QueryRowContext(ctx, qUserExists, p.UserID[:]).Scan(&one)
		if err == nil {
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		_, err = t.ExecContext(ctx, qInsertUser, p.UserID[:], p.Email.String(), e.OccurredAt)
		if platform.IsDuplicate(err) {
			return store.ErrEmailTaken
		}
		return err
	case domain.EmailChanged:
		return update(ctx, t, qSetEmail, p.New.String(), e.StreamID, store.ErrEmailTaken)
	case domain.GoogleLinked:
		return update(ctx, t, qSetGoogle, p.Subject, e.StreamID, store.ErrGoogleTaken)
	default:
		return fmt.Errorf("unknown event %T", e.Payload)
	}
}

func update(ctx context.Context, t *sql.Tx, query, value string, id uuid.UUID, taken error) error {
	res, err := t.ExecContext(ctx, query, value, id[:])
	if platform.IsDuplicate(err) {
		return taken
	}
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	// The DSN counts matched rows, so 0 means the user has no row.
	if n == 0 {
		return errors.New("the user isn't registered")
	}
	return nil
}

// ByID implements store.Users.
func (s *Users) ByID(ctx context.Context, id uuid.UUID) (store.User, error) {
	return s.find(ctx, qUserByID, id[:])
}

// ByEmail implements store.Users.
func (s *Users) ByEmail(ctx context.Context, email domain.Email) (store.User, error) {
	return s.find(ctx, qUserByEmail, email.String())
}

// ByGoogleSubject implements store.Users.
func (s *Users) ByGoogleSubject(ctx context.Context, subject string) (store.User, error) {
	return s.find(ctx, qUserBySubject, subject)
}

func (s *Users) find(ctx context.Context, query string, arg any) (store.User, error) {
	var (
		u   store.User
		id  []byte
		sub sql.NullString
	)
	err := s.db.QueryRowContext(ctx, query, arg).Scan(&id, &u.Email, &sub, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return store.User{}, fmt.Errorf("mysql: find user: %w", apperr.ErrNotFound)
	}
	if err != nil {
		return store.User{}, fmt.Errorf("mysql: find user: %w", err)
	}
	if u.ID, err = parseUUID(id); err != nil {
		return store.User{}, fmt.Errorf("mysql: find user: %w", err)
	}
	u.GoogleSubject = sub.String
	return u, nil
}

// LoginCodes keeps login codes in the login_codes table.
type LoginCodes struct {
	db *platform.DB
}

var _ store.LoginCodes = (*LoginCodes)(nil)

// NewLoginCodes returns the code store on db, which needs Migrations.
func NewLoginCodes(db *platform.DB) *LoginCodes { return &LoginCodes{db: db} }

// Add implements store.LoginCodes.
func (s *LoginCodes) Add(ctx context.Context, tx uow.Tx, c store.LoginCode) error {
	t, err := platform.SQLTx(tx)
	if err != nil {
		return err
	}
	var userID []byte
	if c.UserID != uuid.Nil() {
		userID = c.UserID[:]
	}
	_, err = t.ExecContext(ctx, qInsertCode, c.Hash[:], string(c.Purpose), c.Email, userID, c.Next, c.ExpiresAt)
	if platform.IsDuplicate(err) {
		return fmt.Errorf("mysql: add login code: %w", apperr.ErrConflict)
	}
	if err != nil {
		return fmt.Errorf("mysql: add login code: %w", err)
	}
	return nil
}

// Find implements store.LoginCodes.
func (s *LoginCodes) Find(ctx context.Context, hash store.CodeHash, purpose store.Purpose, now time.Time) (store.LoginCode, error) {
	c, err := scanCode(s.db.QueryRowContext(ctx, qUsableCode, hash[:], string(purpose), now))
	if err != nil {
		return store.LoginCode{}, fmt.Errorf("mysql: find login code: %w", err)
	}
	return c, nil
}

// Consume implements store.LoginCodes. The UPDATE is what makes it single
// use: it locks the row, and a concurrent consume waiting on that lock
// re-reads used_at once the first commits, matching nothing.
func (s *LoginCodes) Consume(ctx context.Context, tx uow.Tx, hash store.CodeHash, purpose store.Purpose, now time.Time) (store.LoginCode, error) {
	t, err := platform.SQLTx(tx)
	if err != nil {
		return store.LoginCode{}, err
	}
	res, err := t.ExecContext(ctx, qConsumeCode, now, hash[:], string(purpose), now)
	if err != nil {
		return store.LoginCode{}, fmt.Errorf("mysql: consume login code: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return store.LoginCode{}, fmt.Errorf("mysql: consume login code: %w", err)
	}
	if n != 1 {
		return store.LoginCode{}, store.ErrCodeUnusable
	}
	c, err := scanCode(t.QueryRowContext(ctx, qCodeByHash, hash[:]))
	if err != nil {
		return store.LoginCode{}, fmt.Errorf("mysql: read consumed login code: %w", err)
	}
	return c, nil
}

func scanCode(row *sql.Row) (store.LoginCode, error) {
	var (
		c       store.LoginCode
		hash    []byte
		purpose string
		userID  []byte
	)
	err := row.Scan(&hash, &purpose, &c.Email, &userID, &c.Next, &c.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return store.LoginCode{}, store.ErrCodeUnusable
	}
	if err != nil {
		return store.LoginCode{}, err
	}
	if len(hash) != len(c.Hash) {
		return store.LoginCode{}, fmt.Errorf("code_hash has %d bytes, want %d", len(hash), len(c.Hash))
	}
	copy(c.Hash[:], hash)
	c.Purpose = store.Purpose(purpose)
	if userID != nil {
		if c.UserID, err = parseUUID(userID); err != nil {
			return store.LoginCode{}, err
		}
	}
	return c, nil
}

// DeleteExpired implements store.LoginCodes. It deletes in batches, so no
// one statement holds locks on many rows.
func (s *LoginCodes) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	var total int64
	for {
		res, err := s.db.ExecContext(ctx, qDeleteExpired, now, deleteBatchSize)
		if err != nil {
			return total, fmt.Errorf("mysql: delete expired login codes: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return total, fmt.Errorf("mysql: delete expired login codes: %w", err)
		}
		total += n
		if n < deleteBatchSize {
			return total, nil
		}
	}
}

func parseUUID(b []byte) (uuid.UUID, error) {
	if len(b) != len(uuid.UUID{}) {
		return uuid.UUID{}, fmt.Errorf("want a 16-byte UUID, got %d bytes", len(b))
	}
	return uuid.UUID(b), nil
}
