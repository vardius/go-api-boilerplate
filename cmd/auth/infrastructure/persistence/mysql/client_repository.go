/*
Package mysql holds view model repositories
*/
package mysql

import (
	"context"
	"database/sql"

	"github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

type clientRepository struct {
	db *sql.DB
}

func (r *clientRepository) Get(ctx context.Context, id string) (*persistence.Client, error) {
	row := r.db.QueryRowContext(ctx, `SELECT * FROM clients WHERE id=? LIMIT 1`, id)

	client := &persistence.Client{}

	err := row.Scan(&client.ID, &client.UserID, &client.Secret, &client.Domain, &client.Info)
	switch {
	case err == sql.ErrNoRows:
		return nil, errors.Wrap(err, errors.NOTFOUND, "Client not found")
	case err != nil:
		return nil, errors.Wrap(err, errors.INTERNAL, "Error while scanning clients table")
	default:
		return client, nil
	}
}

func (r *clientRepository) Add(ctx context.Context, client *persistence.Client) error {
	stmt, err := r.db.PrepareContext(ctx, `INSERT INTO clients (id, userId, secret, domain, data) VALUES (?,?,?,?)`)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Invalid client insert query")
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, client.ID, client.UserID, client.Secret, client.Domain, client.Info)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not add client")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not get affected rows")
	}

	if rows != 1 {
		return errors.New(errors.INTERNAL, "Did not add client")
	}

	return nil
}

func (r *clientRepository) Delete(ctx context.Context, id string) error {
	stmt, err := r.db.PrepareContext(ctx, `DELETE FROM clients WHERE id=?`)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Invalid client delete query")
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, id)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not delete client")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Could not get affected rows")
	}

	if rows != 1 {
		return errors.New(errors.INTERNAL, "Did not delete client")
	}

	return nil
}

// NewClientRepository returns mysql view model repository for client
func NewClientRepository(db *sql.DB) persistence.ClientRepository {
	return &clientRepository{db}
}
