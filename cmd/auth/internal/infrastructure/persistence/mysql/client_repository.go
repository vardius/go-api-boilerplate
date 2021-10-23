/*
Package mysql holds view model repositories
*/
package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"gopkg.in/oauth2.v4"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
)

const createClientTableSQL = `
CREATE TABLE IF NOT EXISTS auth_clients
(
    distinct_id  INT          NOT NULL AUTO_INCREMENT,
    id           CHAR(36)     NOT NULL,
    user_id      CHAR(36)     NOT NULL,
    secret       VARCHAR(255) NOT NULL,
    domain       VARCHAR(255) NOT NULL,
    redirect_url TEXT         NOT NULL,
    scope        JSON         NOT NULL,
    PRIMARY KEY (distinct_id),
    UNIQUE KEY id (id),
    INDEX i_user_id (user_id)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8
    COLLATE = utf8_bin;
`

type clientRepository struct {
	cfg *config.Config
	db  *sql.DB
}

// NewClientRepository returns mysql view model repository for client
func NewClientRepository(ctx context.Context, cfg *config.Config, db *sql.DB) (persistence.ClientRepository, error) {
	if _, err := db.ExecContext(ctx, createClientTableSQL); err != nil {
		return nil, apperrors.Wrap(err)
	}

	return &clientRepository{cfg: cfg, db: db}, nil
}

func (r *clientRepository) GetByID(ctx context.Context, id string) (oauth2.ClientInfo, error) {
	c, err := r.Get(ctx, id)

	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	return c, nil
}

func (r *clientRepository) Get(ctx context.Context, id string) (persistence.Client, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, secret, domain, redirect_url, scope FROM auth_clients WHERE id=? LIMIT 1`, id)

	var scope json.RawMessage
	var client Client

	err := row.Scan(&client.ID, &client.UserID, &client.Secret, &client.Domain, &client.RedirectURL, &scope)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, apperrors.Wrap(fmt.Errorf("%w: Client (id:%s) not found: %s", apperrors.ErrNotFound, id, err))
	case err != nil:
		return nil, apperrors.Wrap(err)
	default:
		if err := json.Unmarshal(scope, &client.Scopes); err != nil {
			return nil, apperrors.Wrap(err)
		}
		return &client, nil
	}
}

func (r *clientRepository) FindAllByUserID(ctx context.Context, userID string, limit, offset int64) ([]persistence.Client, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, user_id, secret, domain, redirect_url, scope  FROM auth_clients WHERE user_id=? AND domain!=? ORDER BY distinct_id DESC LIMIT ? OFFSET ?`,
		userID,
		r.cfg.App.Domain,
		limit,
		offset,
	)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	defer rows.Close()

	var clients []persistence.Client

	for rows.Next() {
		var scope json.RawMessage
		var client Client
		if err := rows.Scan(&client.ID, &client.UserID, &client.Secret, &client.Domain, &client.RedirectURL, &scope); err != nil {
			return nil, apperrors.Wrap(err)
		}
		if err := json.Unmarshal(scope, &client.Scopes); err != nil {
			return nil, apperrors.Wrap(err)
		}

		clients = append(clients, &client)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(err)
	}

	return clients, nil
}

func (r *clientRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	var total int64

	row := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(distinct_id) FROM auth_clients WHERE user_id=? AND domain!=?`,
		userID,
		r.cfg.App.Domain,
	)
	if err := row.Scan(&total); err != nil {
		return 0, apperrors.Wrap(err)
	}

	return total, nil
}

func (r *clientRepository) Add(ctx context.Context, c persistence.Client) error {
	client := Client{
		ID:          c.GetID(),
		UserID:      c.GetUserID(),
		Secret:      c.GetSecret(),
		Domain:      c.GetDomain(),
		RedirectURL: c.GetRedirectURL(),
		Scopes:      c.GetScopes(),
	}

	scope, err := json.Marshal(c.GetScopes())
	if err != nil {
		return apperrors.Wrap(err)
	}

	stmt, err := r.db.PrepareContext(ctx, `INSERT INTO auth_clients (id, user_id, secret, domain, redirect_url, scope) VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return apperrors.Wrap(err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, client.ID, client.UserID, client.Secret, client.Domain, client.RedirectURL, scope)
	if err != nil {
		return apperrors.Wrap(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return apperrors.Wrap(err)
	}

	if rows != 1 {
		return apperrors.New("Did not add client")
	}

	return nil
}

func (r *clientRepository) Delete(ctx context.Context, id string) error {
	stmt, err := r.db.PrepareContext(ctx, `DELETE FROM auth_clients WHERE id=?`)
	if err != nil {
		return apperrors.Wrap(err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, id)
	if err != nil {
		return apperrors.Wrap(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return apperrors.Wrap(err)
	}

	if rows != 1 {
		return apperrors.New("Did not delete client")
	}

	return nil
}
