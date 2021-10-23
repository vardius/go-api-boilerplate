package memory

import (
	"context"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"sync"

	"gopkg.in/oauth2.v4"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
)

type clientRepository struct {
	cfg *config.Config
	sync.RWMutex
	clients map[string]persistence.Client
}

// NewClientRepository returns memory view model repository for client
func NewClientRepository(cfg *config.Config) persistence.ClientRepository {
	return &clientRepository{cfg: cfg, clients: make(map[string]persistence.Client)}
}

func (r *clientRepository) GetByID(ctx context.Context, id string) (oauth2.ClientInfo, error) {
	r.RLock()
	defer r.RUnlock()

	v, ok := r.clients[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return v, nil
}

func (r *clientRepository) Get(ctx context.Context, id string) (persistence.Client, error) {
	r.RLock()
	defer r.RUnlock()

	v, ok := r.clients[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return v, nil
}

func (r *clientRepository) FindAllByUserID(ctx context.Context, userID string, limit, offset int64) ([]persistence.Client, error) {
	r.RLock()
	defer r.RUnlock()

	var i int64
	var clients []persistence.Client
	for _, v := range r.clients {
		if i < offset {
			continue
		}
		i++

		if v.GetUserID() != userID {
			continue
		}

		clients = append(clients, v)

		if int64(len(clients)) == limit {
			return clients, nil
		}
	}

	return clients, nil
}

func (r *clientRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	r.RLock()
	defer r.RUnlock()

	var i int64
	for _, v := range r.clients {
		if v.GetUserID() != userID {
			continue
		}
		i++
	}

	return i, nil
}

func (r *clientRepository) Add(ctx context.Context, c persistence.Client) error {
	r.Lock()
	defer r.Unlock()

	r.clients[c.GetID()] = c
	return nil
}

func (r *clientRepository) Delete(ctx context.Context, id string) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.clients[id]; ok {
		delete(r.clients, id)
	}

	return nil
}
