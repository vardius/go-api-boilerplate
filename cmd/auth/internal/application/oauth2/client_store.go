package oauth2

import (
	"context"
	"encoding/json"

	"gopkg.in/oauth2.v4"
	oauth2models "gopkg.in/oauth2.v4/models"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/client"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

// NewClientStore create client store
func NewClientStore(r persistence.ClientRepository, cb commandbus.CommandBus) *ClientStore {
	return &ClientStore{
		repository: r,
		commandBus: cb,
	}
}

// ClientStore client information store
type ClientStore struct {
	repository persistence.ClientRepository
	commandBus commandbus.CommandBus
}

// GetByID according to the UserID for the client information
func (cs *ClientStore) GetByID(ctx context.Context, id string) (oauth2.ClientInfo, error) {
	c, err := cs.repository.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return cs.toClientInfo(c.GetData())
}

// SetInternal set pkg system client information
func (cs *ClientStore) Set(ctx context.Context, info oauth2.ClientInfo) error {
	out := make(chan error, 1)
	defer close(out)

	c := client.Create{
		ClientInfo: info,
	}

	if err := cs.commandBus.Publish(ctx, c); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (cs *ClientStore) toClientInfo(data []byte) (oauth2.ClientInfo, error) {
	info := oauth2models.Client{}
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, errors.Wrap(err)
	}

	return &info, nil
}
