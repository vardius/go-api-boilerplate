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

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
)

type clientRepository struct {
	db *sql.DB
}

func (r *clientRepository) Get(ctx context.Context, id string) (persistence.Client, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, secret, domain, redirect_url, scope FROM clients WHERE id=? LIMIT 1`, id)

	var scope json.RawMessage
	var client Client

	err := row.Scan(&client.ID, &client.UserID, &client.Secret, &client.Domain, &client.RedirectURL, &scope)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, apperrors.Wrap(fmt.Errorf("%w: Client (id:%s) not found: %s", application.ErrNotFound, id, err))
	case err != nil:
		return nil, apperrors.Wrap(err)
	default:
		if err := json.Unmarshal(scope, &client.Scopes); err != nil {
			return nil, apperrors.Wrap(err)
		}
		return &client, nil
	}
}

func (r *clientRepository) GetByUserDomain(ctx context.Context, userID, domain string) (persistence.Client, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, secret, domain, redirect_url, scope FROM clients WHERE user_id=? AND domain=? LIMIT 1`, userID, domain)

	var scope json.RawMessage
	var client Client

	err := row.Scan(&client.ID, &client.UserID, &client.Secret, &client.Domain, &client.RedirectURL, &scope)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, apperrors.Wrap(fmt.Errorf("%w: Client (domain:%s) not found: %s", application.ErrNotFound, domain, err))
	case err != nil:
		return nil, apperrors.Wrap(err)
	default:
		if err := json.Unmarshal(scope, &client.Scopes); err != nil {
			return nil, apperrors.Wrap(err)
		}
		return &client, nil
	}
}

// NewClientRepository returns mysql view model repository for client
func NewClientRepository(db *sql.DB) persistence.ClientRepository {
	return &clientRepository{db}
}
