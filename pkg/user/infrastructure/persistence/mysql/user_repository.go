/*
Package mysql holds view model repositories
*/
package mysql

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/errors"
)

// User holds current state of user
type User struct {
	ID         uuid.UUID `json:"id"`
	Email      string    `json:"email"`
	FacebookID string    `json:"facebookId"`
	GoogleID   string    `json:"googleId"`
}

// UserRepository allows to get/save current state of user to mysql storage
type UserRepository interface {
	FindAll(ctx context.Context) ([]*User, error)
	Get(ctx context.Context, id string) (*User, error)
	Add(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
}

type userRepository struct {
	db *sql.DB
}

func (r *userRepository) FindAll(ctx context.Context) ([]*User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, email, facebookId, googleId FROM users ORDER BY id DESC`)
	if err != nil {
		return nil, errors.Wrap(err, errors.INTERNAL, "Could not query database")
	}
	defer rows.Close()

	users := []*User{}

	for rows.Next() {
		user := &User{}
		err = rows.Scan(&user.ID, &user.Email, &user.FacebookID, &user.GoogleID)
		if err != nil {
			return nil, errors.Wrap(err, errors.INTERNAL, "Error while scanning users table")
		}

		users = append(users, user)
	}

	err = rows.Err()
	if err != nil {
		return nil, errors.Wrap(err, errors.INTERNAL, "Error while getting rows")
	}

	return users, nil
}

func (r *userRepository) Get(ctx context.Context, id string) (*User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, email, facebookId, googleId FROM users WHERE id=?`, id)

	user := &User{}

	err := row.Scan(&user.ID, &user.Email, &user.FacebookID, &user.GoogleID)
	switch {
	case err == sql.ErrNoRows:
		return nil, errors.Wrap(err, errors.NOTFOUND, "User not found")
	case err != nil:
		return nil, errors.Wrap(err, errors.INTERNAL, "Error while scanning users table")
	default:
		return user, nil
	}
}

func (r *userRepository) Add(ctx context.Context, user *User) error {
	stmt, err := r.db.PrepareContext(ctx, "INSERT INTO users(id, email, facebookId, googleId) VALUES(?,?)")
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Invalid user insert query")
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, user.ID, user.Email, user.FacebookID, user.GoogleID)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not add user")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not get affected rows")
	}

	if rows != 1 {
		return errors.Wrap(err, errors.INTERNAL, "Did not add user")
	}

	return nil
}

func (r *userRepository) Update(ctx context.Context, user *User) error {
	stmt, err := r.db.PrepareContext(ctx, "UPDATE users SET email=?, facebookId=?, googleId=? WHERE id=?")
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Invalid user update query")
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, user.Email, user.FacebookID, user.GoogleID, user.ID)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not update user")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not get affected rows")
	}

	if rows != 1 {
		return errors.Wrap(err, errors.INTERNAL, "Did not update user")
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	stmt, err := r.db.PrepareContext(ctx, "DELETE FROM users WHERE id=?")
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Invalid user delete query")
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, id)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not delete user")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not get affected rows")
	}

	if rows != 1 {
		return errors.Wrap(err, errors.INTERNAL, "Did not delete user")
	}

	return nil
}

// NewUserRepository returns mysql view model repository for user
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db}
}
