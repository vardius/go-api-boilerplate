package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

type Provider interface {
	GetByUserEmail(ctx context.Context, userEmail, domain string) (*identity.Identity, error)
}

type identityProvider struct {
	db *sql.DB
}

func NewIdentityProvider(db *sql.DB) *identityProvider {
	return &identityProvider{
		db: db,
	}
}

func (p *identityProvider) GetByUserEmail(ctx context.Context, userEmail, domain string) (*identity.Identity, error) {
	var i identity.Identity

	row := p.db.QueryRowContext(ctx, `
SELECT c.id, c.secret, u.id, u.email_address
FROM clients AS c
  INNER JOIN users AS u ON u.id = c.user_id
WHERE u.email_address = ?
  AND c.domain = ?
LIMIT 1
`, userEmail, domain)

	err := row.Scan(&i.ClientID, &i.ClientSecret, &i.UserID, &i.UserEmail)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, apperrors.Wrap(fmt.Errorf("%w: credentials not found: %s", application.ErrNotFound, err))
	case err != nil:
		return nil, apperrors.Wrap(err)
	}

	return &i, nil
}

func (p *identityProvider) GetByUserID(ctx context.Context, userID, clientID uuid.UUID) (*identity.Identity, error) {
	var i identity.Identity

	row := p.db.QueryRowContext(ctx, `
SELECT c.id, c.secret, c.domain, u.id, u.email_address
FROM clients AS c
  INNER JOIN users AS u ON u.id = c.user_id
WHERE c.id = ?
  AND c.user_id = ?
LIMIT 1
`, clientID, userID)

	err := row.Scan(&i.ClientID, &i.ClientSecret, &i.ClientDomain, &i.UserID, &i.UserEmail)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, apperrors.Wrap(fmt.Errorf("%w: credentials not found: %s", application.ErrNotFound, err))
	case err != nil:
		return nil, apperrors.Wrap(err)
	}

	return &i, nil
}
