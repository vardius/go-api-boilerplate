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
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/mysql"
)

const createUsersTableSQL = `
CREATE TABLE IF NOT EXISTS user_users
(
    distinct_id   INT                                  NOT NULL AUTO_INCREMENT,
    id       	  CHAR(36)                             NOT NULL,
    role       	  SMALLINT                             NOT NULL,
    email_address VARCHAR(255) COLLATE utf8_general_ci NOT NULL,
    facebook_id   VARCHAR(255) DEFAULT NULL,
    google_id     VARCHAR(255) DEFAULT NULL,
    PRIMARY KEY (distinct_id),
    UNIQUE KEY id (id),
    UNIQUE KEY email_address (email_address),
    INDEX i_facebook_id (facebook_id),
    INDEX i_google_id (google_id)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8
    COLLATE = utf8_bin;
`

// NewUserRepository returns mysql view model repository for user
func NewUserRepository(ctx context.Context, db *sql.DB) (persistence.UserRepository, error) {
	if _, err := db.ExecContext(ctx, createUsersTableSQL); err != nil {
		return nil, apperrors.Wrap(err)
	}

	return &userRepository{db}, nil
}

type userRepository struct {
	db *sql.DB
}

func (r *userRepository) FindAll(ctx context.Context, limit, offset int64) ([]persistence.User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, email_address, role, facebook_id, google_id FROM user_users ORDER BY distinct_id ASC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	defer rows.Close()

	var users []persistence.User

	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email, &user.Role, &user.FacebookID, &user.GoogleID); err != nil {
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
	row := r.db.QueryRowContext(ctx, `SELECT id, email_address, role, facebook_id, google_id FROM user_users WHERE id=? LIMIT 1`, id)

	var user User

	err := row.Scan(&user.ID, &user.Email, &user.Role, &user.FacebookID, &user.GoogleID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, apperrors.Wrap(fmt.Errorf("%w: %s", apperrors.ErrNotFound, err))
	case err != nil:
		return nil, apperrors.Wrap(err)
	default:
		return user, nil
	}
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (persistence.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, email_address, role, facebook_id, google_id FROM user_users WHERE email_address=? LIMIT 1`, email)

	var user User

	err := row.Scan(&user.ID, &user.Email, &user.Role, &user.FacebookID, &user.GoogleID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, apperrors.Wrap(fmt.Errorf("%w: %s", apperrors.ErrNotFound, err))
	case err != nil:
		return nil, apperrors.Wrap(err)
	default:
		return user, nil
	}
}

func (r *userRepository) GetByFacebookID(ctx context.Context, facebookID string) (persistence.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, email_address, role, facebook_id, google_id FROM user_users WHERE facebook_id=? LIMIT 1`, facebookID)

	var user User

	err := row.Scan(&user.ID, &user.Email, &user.Role, &user.FacebookID, &user.GoogleID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, apperrors.Wrap(fmt.Errorf("%w: %s", apperrors.ErrNotFound, err))
	case err != nil:
		return nil, apperrors.Wrap(err)
	default:
		return user, nil
	}
}

func (r *userRepository) GetByGoogleID(ctx context.Context, googleID string) (persistence.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, email_address, role, facebook_id, google_id FROM user_users WHERE google_id=? LIMIT 1`, googleID)

	var user User

	err := row.Scan(&user.ID, &user.Email, &user.Role, &user.FacebookID, &user.GoogleID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, apperrors.Wrap(fmt.Errorf("%w: %s", apperrors.ErrNotFound, err))
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

	stmt, err := r.db.PrepareContext(ctx, `INSERT IGNORE INTO user_users (id, email_address, role, facebook_id, google_id) VALUES (?,?,?,?,?)`)
	if err != nil {
		return apperrors.Wrap(err)
	}
	defer stmt.Close()

	if _, err := stmt.ExecContext(ctx, user.ID, user.Email, user.Role, user.FacebookID, user.GoogleID); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

func (r *userRepository) UpdateEmail(ctx context.Context, id, email string) error {
	stmt, err := r.db.PrepareContext(ctx, `UPDATE user_users SET email_address=? WHERE id=?`)
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
		return apperrors.New("did not update user")
	}

	return nil
}

func (r *userRepository) UpdateFacebookID(ctx context.Context, id, facebookID string) error {
	stmt, err := r.db.PrepareContext(ctx, `UPDATE user_users SET facebook_id=? WHERE id=?`)
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
	stmt, err := r.db.PrepareContext(ctx, `UPDATE users SET google_id=? WHERE id=?`)
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
	stmt, err := r.db.PrepareContext(ctx, `DELETE FROM user_users WHERE id=?`)
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

func (r *userRepository) Count(ctx context.Context) (int64, error) {
	var totalUsers int64

	row := r.db.QueryRowContext(ctx, `SELECT COUNT(distinct_id) FROM user_users`)
	if err := row.Scan(&totalUsers); err != nil {
		return 0, apperrors.Wrap(err)
	}

	return totalUsers, nil
}
