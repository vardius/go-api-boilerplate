/*
Package mysql holds view model repositories
*/
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
)

// NewUserRepository returns mysql view model repository for user
func NewUserRepository(db *sql.DB) persistence.UserRepository {
	return &userRepository{db}
}

type userRepository struct {
	db *sql.DB
}

func (r *userRepository) Get(ctx context.Context, id string) (persistence.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, email_address FROM users WHERE id=? LIMIT 1`, id)

	var user User
	if err := row.Scan(&user.ID, &user.Email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.Wrap(fmt.Errorf("%w: %s", application.ErrNotFound, err))
		}

		return nil, apperrors.Wrap(err)
	}

	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (persistence.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id FROM users WHERE email_address=? LIMIT 1`, email)

	var user User
	if err := row.Scan(&user.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.Wrap(fmt.Errorf("%w: %s", application.ErrNotFound, err))
		}

		return nil, apperrors.Wrap(err)
	}

	user.Email = email

	return user, nil
}
