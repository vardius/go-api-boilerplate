/*
Package mysql holds view model repositories
*/
package mysql

import (
	"context"
	"database/sql"

	"github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

// NewUserRepository returns mysql view model repository for user
func NewUserRepository(db *sql.DB) persistence.UserRepository {
	return &userRepository{db}
}

type userRepository struct {
	db *sql.DB
}

func (r *userRepository) FindAll(ctx context.Context, limit, offset int32) ([]persistence.User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, emailAddress, facebookId, googleId FROM users ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, errors.INTERNAL, "Could not query database")
	}
	defer rows.Close()

	var users []persistence.User

	for rows.Next() {
		user := User{}
		err = rows.Scan(&user.ID, &user.Email, &user.FacebookID, &user.GoogleID)
		if err != nil {
			return nil, errors.Wrap(err, errors.INTERNAL, "Error while scanning users table")
		}

		users = append(users, user)
	}

	err = rows.Err()
	if err != nil {
		return nil, errors.New(errors.INTERNAL, "Error while getting rows")
	}

	return users, nil
}

func (r *userRepository) Get(ctx context.Context, id string) (persistence.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, emailAddress, facebookId, googleId FROM users WHERE id=? LIMIT 1`, id)

	user := User{}

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

func (r *userRepository) Add(ctx context.Context, u persistence.User) error {
	user, ok := u.(User)
	if !ok {
		return errors.New(errors.INTERNAL, "Could not parse interface to mysql type")
	}

	stmt, err := r.db.PrepareContext(ctx, `INSERT INTO users (id, emailAddress, facebookId, googleId) VALUES (?,?,?,?)`)
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
		return errors.New(errors.INTERNAL, "Did not add user")
	}

	return nil
}

func (r *userRepository) UpdateEmail(ctx context.Context, id, email string) error {
	stmt, err := r.db.PrepareContext(ctx, `UPDATE users SET emailAddress=? WHERE id=?`)
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
		return errors.New(errors.INTERNAL, "Did not update user")
	}

	return nil
}

func (r *userRepository) UpdateFacebookID(ctx context.Context, id, facebookID string) error {
	stmt, err := r.db.PrepareContext(ctx, `UPDATE users SET facebookID=? WHERE id=?`)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Invalid user update query")
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, facebookID, id)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not update user")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not get affected rows")
	}

	if rows != 1 {
		return errors.New(errors.INTERNAL, "Did not update user")
	}

	return nil
}

func (r *userRepository) UpdateGoogleID(ctx context.Context, id, googleID string) error {
	stmt, err := r.db.PrepareContext(ctx, `UPDATE users SET googleID=? WHERE id=?`)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Invalid user update query")
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, googleID, id)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not update user")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not get affected rows")
	}

	if rows != 1 {
		return errors.New(errors.INTERNAL, "Did not update user")
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	stmt, err := r.db.PrepareContext(ctx, `DELETE FROM users WHERE id=?`)
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
		return errors.New(errors.INTERNAL, "Did not delete user")
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

	return totalUsers, nil
}
