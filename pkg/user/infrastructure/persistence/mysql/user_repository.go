/*
Package mysql holds view model repositories
*/
package mysql

import (
	"context"
	"database/sql"

	"github.com/vardius/go-api-boilerplate/pkg/common/application/errors"
	"github.com/vardius/go-api-boilerplate/pkg/user/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/user/infrastructure/proto"
)

type userRepository struct {
	db *sql.DB
}

func (r *userRepository) FindAll(ctx context.Context, limit, offset int32) ([]*proto.User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, email, facebookId, googleId FROM users ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, errors.INTERNAL, "Could not query database")
	}
	defer rows.Close()

	users := []*proto.User{}

	for rows.Next() {
		user := &proto.User{}
		err = rows.Scan(&user.Id, &user.Email, &user.FacebookId, &user.GoogleId)
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

func (r *userRepository) Get(ctx context.Context, id string) (*proto.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, email, facebookId, googleId FROM users WHERE id=?`, id)

	user := &proto.User{}

	err := row.Scan(&user.Id, &user.Email, &user.FacebookId, &user.GoogleId)
	switch {
	case err == sql.ErrNoRows:
		return nil, errors.Wrap(err, errors.NOTFOUND, "User not found")
	case err != nil:
		return nil, errors.Wrap(err, errors.INTERNAL, "Error while scanning users table")
	default:
		return user, nil
	}
}

func (r *userRepository) Add(ctx context.Context, user *proto.User) error {
	stmt, err := r.db.PrepareContext(ctx, "INSERT INTO users(id, email, facebookId, googleId) VALUES(?,?)")
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Invalid user insert query")
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, user.Id, user.Email, user.FacebookId, user.GoogleId)
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

func (r *userRepository) UpdateEmail(ctx context.Context, id, email string) error {
	stmt, err := r.db.PrepareContext(ctx, "UPDATE users SET email=? WHERE id=?")
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Invalid user update query")
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, email, id)
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

func (r *userRepository) Count(ctx context.Context) (int32, error) {
	var totalUsers int32

	row := r.db.QueryRowContext(ctx, `SELECT COUNT(distinctId) FROM users`)
	err := row.Scan(&totalUsers)
	if err != nil {
		return 0, errors.Wrap(err, errors.INTERNAL, "Could not count users")
	}

	return 0, nil
}

// NewUserRepository returns mysql view model repository for user
func NewUserRepository(db *sql.DB) persistence.UserRepository {
	return &userRepository{db}
}
