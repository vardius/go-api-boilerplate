// Package mysql keeps the auth service's store in MySQL 9.7.
//
// Migrate creates its tables, under their own goose version table. A
// single-use operation locks its row with SELECT … FOR UPDATE inside a
// transaction, so a concurrent use waits for it and then finds the row
// already used. Expired rows are deleted by Purge, which the service runs
// periodically with store.PurgeWorker.
package mysql

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	platform "github.com/vardius/go-api-boilerplate/internal/platform/mysql"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// MigrationsTable is the goose version table of the auth service's
// migrations, apart from the platform's.
const MigrationsTable = "goose_auth"

// The tables, as the migrations create them.
const (
	clientsTable  = "auth_clients"
	grantsTable   = "auth_grants"
	tokensTable   = "auth_refresh_tokens"
	requestsTable = "auth_authorization_requests"
	codesTable    = "auth_authorization_codes"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migrations creates the auth service's tables.
var Migrations = func() fs.FS {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err) // The directory is embedded above, so this can't happen.
	}
	return sub
}()

// Migrate applies the migrations db hasn't run yet.
func Migrate(ctx context.Context, db *platform.DB) error {
	return platform.Migrate(ctx, db.DB, Migrations, MigrationsTable)
}

// Store is a store.Store in MySQL.
type Store struct {
	db *platform.DB
}

var (
	_ store.Store  = (*Store)(nil)
	_ store.Purger = (*Store)(nil)
)

// New returns the store on db, which needs Migrate.
func New(db *platform.DB) *Store { return &Store{db: db} }

func (s *Store) tx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	return s.db.Do(ctx, func(tx uow.Tx) error {
		t, err := platform.SQLTx(tx)
		if err != nil {
			return err
		}
		return fn(t)
	})
}

// execer is what both *sql.DB and *sql.Tx run statements with.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func list(v []string) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}

// jsonList scans a JSON array of strings.
type jsonList struct{ dst *[]string }

func (c jsonList) Scan(src any) error {
	b, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("scan a JSON list from %T", src)
	}
	return json.Unmarshal(b, c.dst)
}

// uuidColumn scans a BINARY(16) column; NULL is uuid.Nil().
type uuidColumn struct{ dst *uuid.UUID }

func (c uuidColumn) Scan(src any) error {
	if src == nil {
		*c.dst = uuid.Nil()
		return nil
	}
	b, ok := src.([]byte)
	if !ok || len(b) != len(c.dst) {
		return fmt.Errorf("scan a UUID from %T of %d bytes", src, len(b))
	}
	copy(c.dst[:], b)
	return nil
}

func insertError(what string, err error) error {
	if platform.IsDuplicate(err) {
		return fmt.Errorf("mysql: %s: %w: %w", what, apperr.ErrConflict, err)
	}
	return fmt.Errorf("mysql: %s: %w", what, err)
}

func notFound(what string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("mysql: %s: %w", what, apperr.ErrNotFound)
	}
	return fmt.Errorf("mysql: %s: %w", what, err)
}

const clientColumns = "id, owner_id, name, redirect_uris, scopes, secret_hash, created_at"

type scanner interface{ Scan(dest ...any) error }

func scanClient(row scanner) (store.Client, error) {
	var c store.Client
	err := row.Scan(&c.ID, uuidColumn{&c.OwnerID}, &c.Name, jsonList{&c.RedirectURIs}, jsonList{&c.Scopes}, &c.SecretHash, &c.CreatedAt)
	return c, err
}

// CreateClient implements store.Store.
func (s *Store) CreateClient(ctx context.Context, c store.Client) error {
	uris, err := list(c.RedirectURIs)
	if err != nil {
		return fmt.Errorf("mysql: create client: %w", err)
	}
	scopes, err := list(c.Scopes)
	if err != nil {
		return fmt.Errorf("mysql: create client: %w", err)
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO "+clientsTable+" ("+clientColumns+") VALUES (?, ?, ?, ?, ?, ?, ?)",
		c.ID, c.OwnerID[:], c.Name, uris, scopes, c.SecretHash, c.CreatedAt)
	if err != nil {
		return insertError("create client "+c.ID, err)
	}
	return nil
}

// Client implements store.Store.
func (s *Store) Client(ctx context.Context, id string) (store.Client, error) {
	c, err := scanClient(s.db.QueryRowContext(ctx, "SELECT "+clientColumns+" FROM "+clientsTable+" WHERE id = ?", id))
	if err != nil {
		return store.Client{}, notFound("client "+id, err)
	}
	return c, nil
}

// Clients implements store.Store.
func (s *Store) Clients(ctx context.Context, owner uuid.UUID, after store.ClientKey, limit int) ([]store.Client, error) {
	query, args := "SELECT "+clientColumns+" FROM "+clientsTable+" WHERE owner_id = ?", []any{owner[:]}
	if after != (store.ClientKey{}) {
		// Spelled out rather than as a row comparison, so the range reads
		// plainly against the (owner_id, created_at, id) index.
		query += " AND (created_at > ? OR (created_at = ? AND id > ?))"
		args = append(args, after.CreatedAt, after.CreatedAt, after.ID)
	}
	rows, err := s.db.QueryContext(ctx, query+" ORDER BY created_at, id LIMIT ?", append(args, limit)...)
	if err != nil {
		return nil, fmt.Errorf("mysql: list clients: %w", err)
	}
	defer rows.Close()
	var out []store.Client
	for rows.Next() {
		c, err := scanClient(rows)
		if err != nil {
			return nil, fmt.Errorf("mysql: list clients: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql: list clients: %w", err)
	}
	return out, nil
}

// DeleteClient implements store.Store. Deleting its grants deletes their
// refresh tokens through the foreign key.
func (s *Store) DeleteClient(ctx context.Context, owner uuid.UUID, id string) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "DELETE FROM "+clientsTable+" WHERE id = ? AND owner_id = ?", id, owner[:])
		if err != nil {
			return fmt.Errorf("mysql: delete client %s: %w", id, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("mysql: delete client %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("mysql: delete client %s: %w", id, apperr.ErrNotFound)
		}
		for _, table := range []string{grantsTable, requestsTable, codesTable} {
			if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE client_id = ?", id); err != nil {
				return fmt.Errorf("mysql: delete client %s from %s: %w", id, table, err)
			}
		}
		return nil
	})
}

const grantColumns = "id, user_id, client_id, scopes, user_agent, created_at, last_used_at, expires_at"

func scanGrant(row scanner, extra ...any) (store.Grant, error) {
	var g store.Grant
	dest := append([]any{uuidColumn{&g.ID}, uuidColumn{&g.UserID}, &g.ClientID, jsonList{&g.Scopes}, &g.UserAgent, &g.CreatedAt, &g.LastUsedAt, &g.ExpiresAt}, extra...)
	err := row.Scan(dest...)
	return g, err
}

func insertGrant(ctx context.Context, tx execer, g store.Grant, first store.RefreshToken) error {
	scopes, err := list(g.Scopes)
	if err != nil {
		return fmt.Errorf("mysql: create grant: %w", err)
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO "+grantsTable+" ("+grantColumns+") VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		g.ID[:], g.UserID[:], g.ClientID, scopes, g.UserAgent, g.CreatedAt, g.LastUsedAt, g.ExpiresAt)
	if err != nil {
		return insertError("create grant "+g.ID.String(), err)
	}
	return insertToken(ctx, tx, g.ID, first)
}

func insertToken(ctx context.Context, tx execer, grant uuid.UUID, t store.RefreshToken) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO "+tokensTable+" (hash, grant_id, expires_at) VALUES (?, ?, ?)", t.Hash[:], grant[:], t.ExpiresAt)
	if err != nil {
		return insertError("store a refresh token", err)
	}
	return nil
}

// CreateGrant implements store.Store.
func (s *Store) CreateGrant(ctx context.Context, g store.Grant, first store.RefreshToken) error {
	return s.tx(ctx, func(tx *sql.Tx) error { return insertGrant(ctx, tx, g, first) })
}

// Grant implements store.Store.
func (s *Store) Grant(ctx context.Context, id uuid.UUID, now time.Time) (store.Grant, error) {
	g, err := scanGrant(s.db.QueryRowContext(ctx, "SELECT "+grantColumns+" FROM "+grantsTable+" WHERE id = ? AND expires_at > ?", id[:], now))
	if err != nil {
		return store.Grant{}, notFound("grant "+id.String(), err)
	}
	return g, nil
}

// Grants implements store.Store.
func (s *Store) Grants(ctx context.Context, user uuid.UUID, client string, now time.Time, limit int) ([]store.Grant, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT "+grantColumns+" FROM "+grantsTable+
			" WHERE user_id = ? AND client_id = ? AND expires_at > ? ORDER BY last_used_at DESC, id DESC LIMIT ?",
		user[:], client, now, limit)
	if err != nil {
		return nil, fmt.Errorf("mysql: list grants: %w", err)
	}
	defer rows.Close()
	var out []store.Grant
	for rows.Next() {
		g, err := scanGrant(rows)
		if err != nil {
			return nil, fmt.Errorf("mysql: list grants: %w", err)
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql: list grants: %w", err)
	}
	return out, nil
}

// RevokeGrant implements store.Store.
func (s *Store) RevokeGrant(ctx context.Context, id uuid.UUID) error {
	if _, err := s.db.ExecContext(ctx, "DELETE FROM "+grantsTable+" WHERE id = ?", id[:]); err != nil {
		return fmt.Errorf("mysql: revoke grant %s: %w", id, err)
	}
	return nil
}

// RefreshTokenGrant implements store.Store.
func (s *Store) RefreshTokenGrant(ctx context.Context, h store.Hash, now time.Time) (store.Grant, bool, error) {
	var rotated bool
	row := s.db.QueryRowContext(ctx,
		"SELECT g.id, g.user_id, g.client_id, g.scopes, g.user_agent, g.created_at, g.last_used_at, g.expires_at, t.rotated"+
			" FROM "+tokensTable+" t JOIN "+grantsTable+" g ON g.id = t.grant_id WHERE t.hash = ? AND t.expires_at > ?",
		h[:], now)
	g, err := scanGrant(row, &rotated)
	if err != nil {
		return store.Grant{}, false, notFound("refresh token", err)
	}
	return g, rotated, nil
}

// RotateRefreshToken implements store.Store.
func (s *Store) RotateRefreshToken(ctx context.Context, h store.Hash, next store.RefreshToken, now time.Time) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		var (
			grant   uuid.UUID
			expires time.Time
			rotated bool
		)
		err := tx.QueryRowContext(ctx, "SELECT grant_id, expires_at, rotated FROM "+tokensTable+" WHERE hash = ? FOR UPDATE", h[:]).
			Scan(uuidColumn{&grant}, &expires, &rotated)
		switch {
		case err != nil:
			return notFound("rotate refresh token", err)
		case !now.Before(expires):
			return fmt.Errorf("mysql: rotate refresh token: expired: %w", apperr.ErrNotFound)
		case rotated:
			return fmt.Errorf("mysql: rotate refresh token: %w", store.ErrUsed)
		}
		if _, err := tx.ExecContext(ctx, "UPDATE "+tokensTable+" SET rotated = TRUE WHERE hash = ?", h[:]); err != nil {
			return fmt.Errorf("mysql: rotate refresh token: %w", err)
		}
		if err := insertToken(ctx, tx, grant, next); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE "+grantsTable+" SET last_used_at = ?, expires_at = ? WHERE id = ?", now, next.ExpiresAt, grant[:]); err != nil {
			return fmt.Errorf("mysql: rotate refresh token: %w", err)
		}
		return nil
	})
}

const requestColumns = "id, user_id, client_id, redirect_uri, scopes, state, code_challenge, expires_at"

func scanRequest(row scanner) (store.AuthorizationRequest, error) {
	var r store.AuthorizationRequest
	err := row.Scan(uuidColumn{&r.ID}, uuidColumn{&r.UserID}, &r.ClientID, &r.RedirectURI, jsonList{&r.Scopes}, &r.State, &r.CodeChallenge, &r.ExpiresAt)
	return r, err
}

// CreateAuthorizationRequest implements store.Store.
func (s *Store) CreateAuthorizationRequest(ctx context.Context, r store.AuthorizationRequest) error {
	scopes, err := list(r.Scopes)
	if err != nil {
		return fmt.Errorf("mysql: create authorization request: %w", err)
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO "+requestsTable+" ("+requestColumns+") VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		r.ID[:], r.UserID[:], r.ClientID, r.RedirectURI, scopes, r.State, r.CodeChallenge, r.ExpiresAt)
	if err != nil {
		return insertError("create authorization request "+r.ID.String(), err)
	}
	return nil
}

const requestWhere = " WHERE id = ? AND user_id = ? AND expires_at > ?"

// AuthorizationRequest implements store.Store.
func (s *Store) AuthorizationRequest(ctx context.Context, id, user uuid.UUID, now time.Time) (store.AuthorizationRequest, error) {
	r, err := scanRequest(s.db.QueryRowContext(ctx, "SELECT "+requestColumns+" FROM "+requestsTable+requestWhere, id[:], user[:], now))
	if err != nil {
		return store.AuthorizationRequest{}, notFound("authorization request "+id.String(), err)
	}
	return r, nil
}

// TakeAuthorizationRequest implements store.Store.
func (s *Store) TakeAuthorizationRequest(ctx context.Context, id, user uuid.UUID, now time.Time) (store.AuthorizationRequest, error) {
	var r store.AuthorizationRequest
	err := s.tx(ctx, func(tx *sql.Tx) error {
		var err error
		r, err = scanRequest(tx.QueryRowContext(ctx, "SELECT "+requestColumns+" FROM "+requestsTable+requestWhere+" FOR UPDATE", id[:], user[:], now))
		if err != nil {
			return notFound("take authorization request "+id.String(), err)
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+requestsTable+" WHERE id = ?", id[:]); err != nil {
			return fmt.Errorf("mysql: take authorization request %s: %w", id, err)
		}
		return nil
	})
	if err != nil {
		return store.AuthorizationRequest{}, err
	}
	return r, nil
}

const codeColumns = "hash, client_id, user_id, redirect_uri, scopes, code_challenge, expires_at, used, grant_id"

// CreateAuthorizationCode implements store.Store.
func (s *Store) CreateAuthorizationCode(ctx context.Context, c store.AuthorizationCode) error {
	scopes, err := list(c.Scopes)
	if err != nil {
		return fmt.Errorf("mysql: create authorization code: %w", err)
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO "+codesTable+" (hash, client_id, user_id, redirect_uri, scopes, code_challenge, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		c.Hash[:], c.ClientID, c.UserID[:], c.RedirectURI, scopes, c.CodeChallenge, c.ExpiresAt)
	if err != nil {
		return insertError("create authorization code", err)
	}
	return nil
}

// AuthorizationCode implements store.Store.
func (s *Store) AuthorizationCode(ctx context.Context, h store.Hash) (store.AuthorizationCode, error) {
	var (
		c    store.AuthorizationCode
		hash []byte
	)
	err := s.db.QueryRowContext(ctx, "SELECT "+codeColumns+" FROM "+codesTable+" WHERE hash = ?", h[:]).
		Scan(&hash, &c.ClientID, uuidColumn{&c.UserID}, &c.RedirectURI, jsonList{&c.Scopes}, &c.CodeChallenge, &c.ExpiresAt, &c.Used, uuidColumn{&c.GrantID})
	if err != nil {
		return store.AuthorizationCode{}, notFound("authorization code", err)
	}
	c.Hash = store.Hash(hash)
	return c, nil
}

// lockUnusedCode locks the code with hash h, failing when it is unknown or
// used.
func lockUnusedCode(ctx context.Context, tx *sql.Tx, h store.Hash) error {
	var used bool
	if err := tx.QueryRowContext(ctx, "SELECT used FROM "+codesTable+" WHERE hash = ? FOR UPDATE", h[:]).Scan(&used); err != nil {
		return notFound("use authorization code", err)
	}
	if used {
		return fmt.Errorf("mysql: use authorization code: %w", store.ErrUsed)
	}
	return nil
}

// SpendAuthorizationCode implements store.Store.
func (s *Store) SpendAuthorizationCode(ctx context.Context, h store.Hash) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		if err := lockUnusedCode(ctx, tx, h); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE "+codesTable+" SET used = TRUE WHERE hash = ?", h[:]); err != nil {
			return fmt.Errorf("mysql: spend authorization code: %w", err)
		}
		return nil
	})
}

// RedeemAuthorizationCode implements store.Store.
func (s *Store) RedeemAuthorizationCode(ctx context.Context, h store.Hash, g store.Grant, first store.RefreshToken) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		if err := lockUnusedCode(ctx, tx, h); err != nil {
			return err
		}
		if err := insertGrant(ctx, tx, g, first); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE "+codesTable+" SET used = TRUE, grant_id = ? WHERE hash = ?", g.ID[:], h[:]); err != nil {
			return fmt.Errorf("mysql: redeem authorization code: %w", err)
		}
		return nil
	})
}

// purgeBatch bounds each DELETE, so a purge never holds many locks at once.
const purgeBatch = 1000

// Purge implements store.Purger. Deleting a grant deletes its tokens.
func (s *Store) Purge(ctx context.Context, now time.Time) error {
	purges := []struct {
		table  string
		before time.Time
	}{
		{grantsTable, now},
		{tokensTable, now},
		{requestsTable, now},
		{codesTable, now.Add(-store.UsedCodeRetention)},
	}
	for _, p := range purges {
		for {
			res, err := s.db.ExecContext(ctx, "DELETE FROM "+p.table+" WHERE expires_at <= ? LIMIT ?", p.before, purgeBatch)
			if err != nil {
				return fmt.Errorf("mysql: purge %s: %w", p.table, err)
			}
			n, err := res.RowsAffected()
			if err != nil {
				return fmt.Errorf("mysql: purge %s: %w", p.table, err)
			}
			if n < purgeBatch {
				break
			}
		}
	}
	return nil
}
