/*
Package mysql holds view model repositories
*/
package mysql

import (
	"context"
	"database/sql"

	"github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/mysql"
)

// NewTokenRepository returns mysql view model repository for token
func NewTokenRepository(db *sql.DB) persistence.TokenRepository {
	return &tokenRepository{db}
}

type tokenRepository struct {
	db *sql.DB
}

func (r *tokenRepository) Get(ctx context.Context, id string) (persistence.Token, error) {
	row := r.db.QueryRowContext(ctx, `SELECT * FROM auth_tokens WHERE id=? LIMIT 1`, id)

	return r.getTokenFromRow(row)
}

func (r *tokenRepository) GetByCode(ctx context.Context, code string) (persistence.Token, error) {
	row := r.db.QueryRowContext(ctx, `SELECT * FROM auth_tokens WHERE code=? LIMIT 1`, code)

	return r.getTokenFromRow(row)
}

func (r *tokenRepository) GetByAccess(ctx context.Context, access string) (persistence.Token, error) {
	row := r.db.QueryRowContext(ctx, `SELECT * FROM auth_tokens WHERE access=? LIMIT 1`, access)

	return r.getTokenFromRow(row)
}

func (r *tokenRepository) GetByRefresh(ctx context.Context, refresh string) (persistence.Token, error) {
	row := r.db.QueryRowContext(ctx, `SELECT * FROM auth_tokens WHERE refresh=? LIMIT 1`, refresh)

	return r.getTokenFromRow(row)
}

func (r *tokenRepository) Add(ctx context.Context, t persistence.Token) error {
	token := Token{
		ID:       t.GetID(),
		ClientID: t.GetClientID(),
		UserID:   t.GetUserID(),
		Scope:    t.GetScope(),
		Access:   t.GetAccess(),
		Refresh:  t.GetRefresh(),
		Code: mysql.NullString{NullString: sql.NullString{
			String: t.GetCode(),
			Valid:  t.GetCode() != "",
		}},
		Data: t.GetData(),
	}

	stmt, err := r.db.PrepareContext(ctx, `INSERT INTO auth_tokens (id, clientId, userId, code, access, refresh, data) VALUES (?,?,?,?,?,?,?)`)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Invalid token insert query")
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, token.ID, token.ClientID, token.UserID, token.Code, token.Access, token.Refresh, token.Data)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not add token")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not get affected rows")
	}

	if rows != 1 {
		return errors.New(errors.INTERNAL, "Did not add token")
	}

	return nil
}

func (r *tokenRepository) Delete(ctx context.Context, id string) error {
	stmt, err := r.db.PrepareContext(ctx, `DELETE FROM auth_tokens WHERE id=?`)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Invalid token delete query")
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, id)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not delete token")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not get affected rows")
	}

	if rows != 1 {
		return errors.New(errors.INTERNAL, "Did not delete token")
	}

	return nil
}

func (r *tokenRepository) getTokenFromRow(row *sql.Row) (persistence.Token, error) {
	token := Token{}
	err := row.Scan(&token.ID, &token.ClientID, &token.UserID, &token.Code, &token.Access, &token.Refresh, &token.Data)

	switch {
	case err == sql.ErrNoRows:
		return nil, errors.Wrap(err, errors.NOTFOUND, "Token not found")
	case err != nil:
		return nil, errors.Wrap(err, errors.INTERNAL, "Error while scanning auth_tokens table")
	default:
		return token, nil
	}
}
