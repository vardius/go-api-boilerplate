/*
Package mysql holds view model repositories
*/
package mysql

import (
	"context"
	"database/sql"
	systemErrors "errors"
	"fmt"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/application"
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
	row := r.db.QueryRowContext(ctx, `SELECT id, client_id, user_id, code, access, refresh FROM auth_tokens WHERE id=? LIMIT 1`, id)

	return r.getTokenFromRow(row)
}

func (r *tokenRepository) GetByCode(ctx context.Context, code string) (persistence.Token, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, client_id, user_id, code, access, refresh FROM auth_tokens WHERE code=? LIMIT 1`, code)

	return r.getTokenFromRow(row)
}

func (r *tokenRepository) GetByAccess(ctx context.Context, access string) (persistence.Token, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, client_id, user_id, code, access, refresh FROM auth_tokens WHERE access=? LIMIT 1`, access)

	return r.getTokenFromRow(row)
}

func (r *tokenRepository) GetByRefresh(ctx context.Context, refresh string) (persistence.Token, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, client_id, user_id, code, access, refresh FROM auth_tokens WHERE refresh=? LIMIT 1`, refresh)

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
	}

	stmt, err := r.db.PrepareContext(ctx, `INSERT INTO auth_tokens (id, client_id, user_id, code, access, refresh) VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return errors.Wrap(fmt.Errorf("%w: Invalid token insert query: %s", application.ErrInternal, err))
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, token.ID, token.ClientID, token.UserID, token.Code, token.Access, token.Refresh)
	if err != nil {
		return errors.Wrap(fmt.Errorf("%w: Could not add token: %s", application.ErrInternal, err))
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(fmt.Errorf("%w: Could not get affected rows: %s", application.ErrInternal, err))
	}

	if rows != 1 {
		return errors.Wrap(fmt.Errorf("%w: Did not add token", application.ErrInternal))
	}

	return nil
}

func (r *tokenRepository) Delete(ctx context.Context, id string) error {
	stmt, err := r.db.PrepareContext(ctx, `DELETE FROM auth_tokens WHERE id=?`)
	if err != nil {
		return errors.Wrap(fmt.Errorf("invalid token delete query: %w", err))
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, id)
	if err != nil {
		return errors.Wrap(fmt.Errorf("could not delete token: %w", err))
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(fmt.Errorf("could not get affected rows: %w", err))
	}

	if rows != 1 {
		return errors.Wrap(fmt.Errorf("did not delete token"))
	}

	return nil
}

func (r *tokenRepository) getTokenFromRow(row *sql.Row) (persistence.Token, error) {
	token := Token{}
	err := row.Scan(&token.ID, &token.ClientID, &token.UserID, &token.Code, &token.Access, &token.Refresh)

	switch {
	case systemErrors.Is(err, sql.ErrNoRows):
		return nil, errors.Wrap(fmt.Errorf("%w: Token not found: %s", application.ErrInternal, err))
	case err != nil:
		return nil, errors.Wrap(fmt.Errorf("%w: Error while scanning auth_tokens table: %s", application.ErrInternal, err))
	default:
		return token, nil
	}
}

func (r *tokenRepository) FindAllByUserID(ctx context.Context, clientID, userID string, limit, offset int32) ([]persistence.Token, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, code, access, refresh FROM auth_tokens WHERE client_id=? AND user_id=? LIMIT ? OFFSET ?`,
		clientID,
		userID,
		limit,
		offset)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	defer rows.Close()

	var tokens []persistence.Token

	for rows.Next() {
		token := Token{}
		if err := rows.Scan(&token.ID, &token.Code, &token.Access, &token.Refresh); err != nil {
			return nil, errors.Wrap(err)
		}

		tokens = append(tokens, token)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err)
	}

	return tokens, nil
}

func (r *tokenRepository) CountByUserID(ctx context.Context, clientID, userID string) (int32, error) {
	var total int32

	row := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(distinct_id) FROM auth_tokens WHERE client_id=? AND user_id=?`,
		clientID,
		userID,
	)
	if err := row.Scan(&total); err != nil {
		return 0, errors.Wrap(err)
	}

	return total, nil
}
