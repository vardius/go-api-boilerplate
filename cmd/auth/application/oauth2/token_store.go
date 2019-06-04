package oauth2

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/cmd/auth/domain/token"
	"github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	oauth2 "gopkg.in/oauth2.v3"
)

// NewTokenStore create a token store instance
func NewTokenStore(r persistence.TokenRepository, cb commandbus.CommandBus) *TokenStore {
	return &TokenStore{r, cb}
}

// TokenStore token storage
type TokenStore struct {
	repository persistence.TokenRepository
	commandBus commandbus.CommandBus
}

// Create create and store the new token information
func (ts *TokenStore) Create(info oauth2.TokenInfo) error {
	ctx := context.Background()
	out := make(chan error)
	defer close(out)

	c := &token.Create{
		TokenInfo: info,
	}

	go func() {
		ts.commandBus.Publish(ctx, fmt.Sprintf("%T", c), c, out)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-out:
		return err
	}
}

// RemoveByCode use the authorization code to delete the token information
func (ts *TokenStore) RemoveByCode(code string) error {
	ctx := context.Background()
	t, err := ts.repository.GetByCode(ctx, code)
	if err != nil {
		return err
	}

	return ts.remove(ctx, t)
}

// RemoveByAccess use the access token to delete the token information
func (ts *TokenStore) RemoveByAccess(access string) error {
	ctx := context.Background()
	t, err := ts.repository.GetByAccess(ctx, access)
	if err != nil {
		return err
	}

	return ts.remove(ctx, t)
}

// RemoveByRefresh use the refresh token to delete the token information
func (ts *TokenStore) RemoveByRefresh(refresh string) error {
	ctx := context.Background()
	t, err := ts.repository.GetByRefresh(ctx, refresh)
	if err != nil {
		return err
	}

	return ts.remove(ctx, t)
}

// GetByCode use the authorization code for token information data
func (ts *TokenStore) GetByCode(code string) (oauth2.TokenInfo, error) {
	t, err := ts.repository.GetByCode(context.Background(), code)
	if err != nil {
		return nil, err
	}

	return t.Info, nil
}

// GetByAccess use the access token for token information data
func (ts *TokenStore) GetByAccess(access string) (oauth2.TokenInfo, error) {
	t, err := ts.repository.GetByAccess(context.Background(), access)
	if err != nil {
		return nil, err
	}

	return t.Info, nil
}

// GetByRefresh use the refresh token for token information data
func (ts *TokenStore) GetByRefresh(refresh string) (oauth2.TokenInfo, error) {
	t, err := ts.repository.GetByRefresh(context.Background(), refresh)
	if err != nil {
		return nil, err
	}

	return t.Info, nil
}

func (ts *TokenStore) remove(ctx context.Context, t *persistence.Token) error {
	out := make(chan error)
	defer close(out)

	c := &token.Remove{
		ID: uuid.MustParse(t.ID),
	}

	go func() {
		ts.commandBus.Publish(ctx, fmt.Sprintf("%T", c), c, out)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-out:
		return err
	}
}
