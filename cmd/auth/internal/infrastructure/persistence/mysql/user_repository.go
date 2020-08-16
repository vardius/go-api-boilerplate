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
)

// NewUserRepository returns mysql view model repository for user
func NewUserRepository(db *sql.DB) persistence.UserRepository {
	return &userRepository{db}
}

type userRepository struct {
	db *sql.DB
}

func (r *userRepository) Get(ctx context.Context, id string) (persistence.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, emailAddress FROM users WHERE id=? LIMIT 1`, id)

	user := User{}

	err := row.Scan(&user.ID, &user.Email)
	switch {
	case systemErrors.Is(err, sql.ErrNoRows):
		return nil, errors.Wrap(fmt.Errorf("%w: %s", application.ErrNotFound, err))
	case err != nil:
		return nil, errors.Wrap(err)
	default:
		return user, nil
	}
}
