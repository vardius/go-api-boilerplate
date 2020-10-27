/*
Package mysql holds view model repositories
*/
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/mysql"
)

// NewUserRepository returns mysql view model repository for user
func NewUserRepository(db *sql.DB) persistence.UserRepository {
	return &userRepository{db}
}

type userRepository struct {
	db *sql.DB
}

func (r *userRepository) FindAll(ctx context.Context, limit, offset int32) ([]persistence.User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, email_address, facebook_id, google_id FROM users ORDER BY distinct_id ASC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	defer rows.Close()

	var users []persistence.User

	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email, &user.FacebookID, &user.GoogleID); err != nil {
			return nil, apperrors.Wrap(err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(err)
	}

	return users, nil
}

func (r *userRepository) Get(ctx context.Context, id string) (persistence.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, email_address, facebook_id, google_id FROM users WHERE id=? LIMIT 1`, id)

	var user User

	err := row.Scan(&user.ID, &user.Email, &user.FacebookID, &user.GoogleID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, apperrors.Wrap(fmt.Errorf("%w: %s", application.ErrNotFound, err))
	case err != nil:
		return nil, apperrors.Wrap(err)
	default:
		return user, nil
	}
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (persistence.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, email_address, facebook_id, google_id FROM users WHERE email_address=? LIMIT 1`, email)

	var user User

	err := row.Scan(&user.ID, &user.Email, &user.FacebookID, &user.GoogleID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, apperrors.Wrap(fmt.Errorf("%w: %s", application.ErrNotFound, err))
	case err != nil:
		return nil, apperrors.Wrap(err)
	default:
		return user, nil
	}
}

func (r *userRepository) Add(ctx context.Context, u persistence.User) error {
	user := User{
		ID:    u.GetID(),
		Email: u.GetEmail(),
		FacebookID: mysql.NullString{NullString: sql.NullString{
			String: u.GetFacebookID(),
			Valid:  u.GetFacebookID() != "",
		}},
		GoogleID: mysql.NullString{NullString: sql.NullString{
			String: u.GetGoogleID(),
			Valid:  u.GetGoogleID() != "",
		}},
	}

	stmt, err := r.db.PrepareContext(ctx, `INSERT IGNORE INTO users (id, email_address, facebook_id, google_id) VALUES (?,?,?,?)`)
	if err != nil {
		return apperrors.Wrap(err)
	}
	defer stmt.Close()

	if _, err := stmt.ExecContext(ctx, user.ID, user.Email, user.FacebookID, user.GoogleID); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

func (r *userRepository) UpdateEmail(ctx context.Context, id, email string) error {
	stmt, err := r.db.PrepareContext(ctx, `UPDATE users SET email_address=? WHERE id=?`)
	if err != nil {
		return apperrors.Wrap(err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, email, id)
	if err != nil {
		return apperrors.Wrap(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return apperrors.Wrap(err)
	}

	if rows != 1 {
		return apperrors.New("Did not update user")
	}

	return nil
}

func (r *userRepository) UpdateFacebookID(ctx context.Context, id, facebookID string) error {
	stmt, err := r.db.PrepareContext(ctx, `UPDATE users SET facebookID=? WHERE id=?`)
	if err != nil {
		return apperrors.Wrap(err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, facebookID, id)
	if err != nil {
		return apperrors.Wrap(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return apperrors.Wrap(err)
	}

	if rows != 1 {
		return apperrors.New("Did not update user")
	}

	return nil
}

func (r *userRepository) UpdateGoogleID(ctx context.Context, id, googleID string) error {
	stmt, err := r.db.PrepareContext(ctx, `UPDATE users SET googleID=? WHERE id=?`)
	if err != nil {
		return apperrors.Wrap(err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, googleID, id)
	if err != nil {
		return apperrors.Wrap(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return apperrors.Wrap(err)
	}

	if rows != 1 {
		return apperrors.New("Did not update user")
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	stmt, err := r.db.PrepareContext(ctx, `DELETE FROM users WHERE id=?`)
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
		return apperrors.New("Did not delete user")
	}

	return nil
}

func (r *userRepository) Count(ctx context.Context) (int32, error) {
	var totalUsers int32

	row := r.db.QueryRowContext(ctx, `SELECT COUNT(distinct_id) FROM users`)
	if err := row.Scan(&totalUsers); err != nil {
		return 0, apperrors.Wrap(err)
	}

	return totalUsers, nil
}
