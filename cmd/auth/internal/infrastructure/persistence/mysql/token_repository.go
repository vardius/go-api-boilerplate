/*
Package mysql holds view model repositories
*/
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/mysql"
)

const createTokenTableSQL = `
CREATE TABLE IF NOT EXISTS auth_tokens
(
    distinct_id INT      NOT NULL AUTO_INCREMENT,
    id          CHAR(36) NOT NULL,
    client_id   CHAR(36) NOT NULL,
    user_id     CHAR(36) NOT NULL,
    code        VARCHAR(255) DEFAULT NULL,
    access      TEXT     NOT NULL,
    refresh     TEXT         DEFAULT NULL,
    expired_at  DATETIME NOT NULL,
    user_agent  TEXT         DEFAULT NULL,
    data        JSON     NOT NULL,
    PRIMARY KEY (distinct_id),
    INDEX i_userId (user_id),
    INDEX i_code (code)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8
    COLLATE = utf8_bin;
`

// NewTokenRepository returns mysql view model repository for token
func NewTokenRepository(ctx context.Context, db *sql.DB) (persistence.TokenRepository, error) {
	if _, err := db.ExecContext(ctx, createTokenTableSQL); err != nil {
		return nil, apperrors.Wrap(err)
	}

	return &tokenRepository{db}, nil
}

type tokenRepository struct {
	db *sql.DB
}

func (r *tokenRepository) Get(ctx context.Context, id string) (persistence.Token, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, client_id, user_id, code, access, refresh, expired_at, user_agent, data FROM auth_tokens WHERE id=? LIMIT 1`, id)

	return r.getTokenFromRow(row)
}

func (r *tokenRepository) GetByCode(ctx context.Context, code string) (persistence.Token, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, client_id, user_id, code, access, refresh, expired_at, user_agent, data FROM auth_tokens WHERE code=? LIMIT 1`, code)

	return r.getTokenFromRow(row)
}

func (r *tokenRepository) GetByAccess(ctx context.Context, access string) (persistence.Token, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, client_id, user_id, code, access, refresh, expired_at, user_agent, data FROM auth_tokens WHERE access=? LIMIT 1`, access)

	return r.getTokenFromRow(row)
}

func (r *tokenRepository) GetByRefresh(ctx context.Context, refresh string) (persistence.Token, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, client_id, user_id, code, access, refresh, expired_at, user_agent, data FROM auth_tokens WHERE refresh=? LIMIT 1`, refresh)

	return r.getTokenFromRow(row)
}

func (r *tokenRepository) Add(ctx context.Context, t persistence.Token) error {
	ti, err := t.TokenInfo()
	if err != nil {
		return apperrors.Wrap(err)
	}

	var expiredAt time.Time
	if code := ti.GetCode(); code != "" {
		expiredAt = ti.GetCodeCreateAt().Add(ti.GetCodeExpiresIn())
	} else {
		expiredAt = ti.GetAccessCreateAt().Add(ti.GetAccessExpiresIn())

		if refresh := ti.GetRefresh(); refresh != "" {
			expiredAt = ti.GetRefreshCreateAt().Add(ti.GetRefreshExpiresIn())
		}
	}

	stmt, err := r.db.PrepareContext(ctx, `INSERT INTO auth_tokens (id, client_id, user_id, code, access, refresh, expired_at, user_agent, data) VALUES (?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return apperrors.Wrap(fmt.Errorf("%w: Invalid token insert query: %s", apperrors.ErrInternal, err))
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(
		ctx,
		t.GetID(),
		ti.GetClientID(),
		ti.GetUserID(),
		mysql.NullString{NullString: sql.NullString{
			String: ti.GetCode(),
			Valid:  ti.GetCode() != "",
		}},
		ti.GetAccess(),
		mysql.NullString{NullString: sql.NullString{
			String: ti.GetRefresh(),
			Valid:  ti.GetRefresh() != "",
		}},
		expiredAt.UTC(),
		mysql.NullString{NullString: sql.NullString{
			String: t.GetUserAgent(),
			Valid:  t.GetUserAgent() != "",
		}},
		t.GetData(),
	)
	if err != nil {
		return apperrors.Wrap(fmt.Errorf("%w: Could not add token: %s", apperrors.ErrInternal, err))
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return apperrors.Wrap(fmt.Errorf("%w: Could not get affected rows: %s", apperrors.ErrInternal, err))
	}

	if rows != 1 {
		return apperrors.Wrap(fmt.Errorf("%w: Did not add token", apperrors.ErrInternal))
	}

	return nil
}

func (r *tokenRepository) Delete(ctx context.Context, id string) error {
	stmt, err := r.db.PrepareContext(ctx, `DELETE FROM auth_tokens WHERE id=?`)
	if err != nil {
		return apperrors.Wrap(fmt.Errorf("invalid token delete query: %w", err))
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, id)
	if err != nil {
		return apperrors.Wrap(fmt.Errorf("could not delete token: %w", err))
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return apperrors.Wrap(fmt.Errorf("could not get affected rows: %w", err))
	}

	if rows != 1 {
		return apperrors.Wrap(fmt.Errorf("did not delete token"))
	}

	return nil
}

func (r *tokenRepository) getTokenFromRow(row *sql.Row) (persistence.Token, error) {
	var token Token
	err := row.Scan(
		&token.ID,
		&token.ClientID,
		&token.UserID,
		&token.Code,
		&token.Access,
		&token.Refresh,
		&token.ExpiredAt,
		&token.UserAgent,
		&token.Data,
	)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, apperrors.Wrap(fmt.Errorf("%w: Token not found: %s", apperrors.ErrInternal, err))
	case err != nil:
		return nil, apperrors.Wrap(fmt.Errorf("%w: Error while scanning auth_tokens table: %s", apperrors.ErrInternal, err))
	default:
		return &token, nil
	}
}

func (r *tokenRepository) FindAllByClientID(ctx context.Context, clientID string, limit, offset int64) ([]persistence.Token, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, client_id, user_id, code, access, refresh, expired_at, user_agent, data FROM auth_tokens WHERE client_id=? LIMIT ? OFFSET ?`,
		clientID,
		limit,
		offset)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	defer rows.Close()

	var tokens []persistence.Token

	for rows.Next() {
		var token Token
		if err := rows.Scan(
			&token.ID,
			&token.ClientID,
			&token.UserID,
			&token.Code,
			&token.Access,
			&token.Refresh,
			&token.ExpiredAt,
			&token.UserAgent,
			&token.Data,
		); err != nil {
			return nil, apperrors.Wrap(err)
		}

		tokens = append(tokens, &token)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(err)
	}

	return tokens, nil
}

func (r *tokenRepository) CountByClientID(ctx context.Context, clientID string) (int64, error) {
	var total int64

	row := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(distinct_id) FROM auth_tokens WHERE client_id=?`,
		clientID,
	)
	if err := row.Scan(&total); err != nil {
		return 0, apperrors.Wrap(err)
	}

	return total, nil
}
