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

type clientRepository struct {
	db *sql.DB
}

func (r *clientRepository) Get(ctx context.Context, id string) (persistence.Client, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, secret, domain, data FROM clients WHERE id=? LIMIT 1`, id)

	client := Client{}

	err := row.Scan(&client.ID, &client.UserID, &client.Secret, &client.Domain, &client.Data)
	switch {
	case systemErrors.Is(err, sql.ErrNoRows):
		return nil, errors.Wrap(fmt.Errorf("%w: Client (id:%s) not found: %s", application.ErrNotFound, id, err))
	case err != nil:
		return nil, errors.Wrap(err)
	default:
		return client, nil
	}
}

func (r *clientRepository) Add(ctx context.Context, c persistence.Client) error {
	client := Client{
		ID:     c.GetID(),
		UserID: c.GetUserID(),
		Secret: c.GetSecret(),
		Domain: c.GetDomain(),
		Data:   c.GetData(),
	}

	stmt, err := r.db.PrepareContext(ctx, `INSERT INTO clients (id, user_id, secret, domain, data) VALUES (?,?,?,?,?)`)
	if err != nil {
		return errors.Wrap(err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, client.ID, client.UserID, client.Secret, client.Domain, client.Data)
	if err != nil {
		return errors.Wrap(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err)
	}

	if rows != 1 {
		return errors.New("Did not add client")
	}

	return nil
}

func (r *clientRepository) Delete(ctx context.Context, id string) error {
	stmt, err := r.db.PrepareContext(ctx, `DELETE FROM clients WHERE id=?`)
	if err != nil {
		return errors.Wrap(err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, id)
	if err != nil {
		return errors.Wrap(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err)
	}

	if rows != 1 {
		return errors.New("Did not delete client")
	}

	return nil
}

// NewClientRepository returns mysql view model repository for client
func NewClientRepository(db *sql.DB) persistence.ClientRepository {
	return &clientRepository{db}
}
