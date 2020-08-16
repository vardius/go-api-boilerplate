package oauth2

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gopkg.in/oauth2.v4"
	oauth2models "gopkg.in/oauth2.v4/models"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
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
func (ts *TokenStore) Create(ctx context.Context, info oauth2.TokenInfo) error {
	out := make(chan error, 1)
	defer close(out)

	c := token.Create{
		TokenInfo: info,
	}

	go func() {
		out <- ts.commandBus.Publish(ctx, c)
	}()

	ctxDoneCh := ctx.Done()
	select {
	case <-ctxDoneCh:
		return errors.Wrap(fmt.Errorf("%w: %s", application.ErrTimeout, ctx.Err()))
	case err := <-out:
		if err != nil {
			return errors.Wrap(fmt.Errorf("create token failed: %w", err))
		}
		return nil
	}
}

// RemoveByCode use the authorization code to delete the token information
func (ts *TokenStore) RemoveByCode(ctx context.Context, code string) error {
	t, err := ts.repository.GetByCode(ctx, code)
	if err != nil {
		return errors.Wrap(err)
	}

	return ts.remove(ctx, t)
}

// RemoveByAccess use the access token to delete the token information
func (ts *TokenStore) RemoveByAccess(ctx context.Context, access string) error {
	t, err := ts.repository.GetByAccess(ctx, access)
	if err != nil {
		return errors.Wrap(err)
	}

	return ts.remove(ctx, t)
}

// RemoveByRefresh use the refresh token to delete the token information
func (ts *TokenStore) RemoveByRefresh(ctx context.Context, refresh string) error {
	t, err := ts.repository.GetByRefresh(ctx, refresh)
	if err != nil {
		return errors.Wrap(err)
	}

	return ts.remove(ctx, t)
}

// GetByCode use the authorization code for token information data
func (ts *TokenStore) GetByCode(ctx context.Context, code string) (oauth2.TokenInfo, error) {
	t, err := ts.repository.GetByCode(ctx, code)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return ts.toTokenInfo(ctx, t.GetData())
}

// GetByAccess use the access token for token information data
func (ts *TokenStore) GetByAccess(ctx context.Context, access string) (oauth2.TokenInfo, error) {
	t, err := ts.repository.GetByAccess(ctx, access)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return ts.toTokenInfo(ctx, t.GetData())
}

// GetByRefresh use the refresh token for token information data
func (ts *TokenStore) GetByRefresh(ctx context.Context, refresh string) (oauth2.TokenInfo, error) {
	t, err := ts.repository.GetByRefresh(ctx, refresh)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return ts.toTokenInfo(ctx, t.GetData())
}

func (ts *TokenStore) toTokenInfo(ctx context.Context, data []byte) (oauth2.TokenInfo, error) {
	info := oauth2models.Token{}
	err := json.Unmarshal(data, &info)
	if err != nil {
		return nil, errors.Wrap(fmt.Errorf("unmarshal token failed: %w", err))
	}

	return &info, nil
}

func (ts *TokenStore) remove(ctx context.Context, t persistence.Token) error {
	out := make(chan error, 1)
	defer close(out)

	id, err := uuid.Parse(t.GetID())
	if err != nil {
		return errors.Wrap(err)
	}

	c := token.Remove{
		ID: id,
	}

	go func() {
		out <- ts.commandBus.Publish(ctx, c)
	}()

	ctxDoneCh := ctx.Done()
	select {
	case <-ctxDoneCh:
		return errors.Wrap(fmt.Errorf("%w: %s", application.ErrTimeout, ctx.Err()))
	case err := <-out:
		if err != nil {
			return errors.Wrap(fmt.Errorf("token remove failed: %w", err))
		}
		return nil
	}
}
