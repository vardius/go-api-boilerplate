package oauth2

import (
	"context"
	"errors"
	"sync"

	"github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/persistence"
	oauth2 "gopkg.in/oauth2.v3"
)

// NewClientStore create client store
func NewClientStore(repository persistence.ClientRepository) *ClientStore {
	return &ClientStore{
		repository: repository,
		internal:   make(map[string]oauth2.ClientInfo),
	}
}

// ClientStore client information store
type ClientStore struct {
	sync.RWMutex
	internal   map[string]oauth2.ClientInfo
	repository persistence.ClientRepository
}

// GetByID according to the ID for the client information
func (cs *ClientStore) GetByID(id string) (oauth2.ClientInfo, error) {
	i, err := cs.getInternal(id)
	if err == nil {
		return i, nil
	}

	c, err := cs.repository.Get(context.Background(), id)
	if err != nil {
		return nil, err
	}

	return c.Info, nil
}

// GetByID according to the ID for the client information
func (cs *ClientStore) getInternal(id string) (cli oauth2.ClientInfo, err error) {
	cs.RLock()
	defer cs.RUnlock()
	if c, ok := cs.internal[id]; ok {
		cli = c
		return
	}
	err = errors.New("not found")
	return
}

// SetInternal set internal system client information
func (cs *ClientStore) SetInternal(id string, cli oauth2.ClientInfo) (err error) {
	cs.Lock()
	defer cs.Unlock()
	cs.internal[id] = cli
	return
}
