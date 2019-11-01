package oauth2

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	oauth2 "gopkg.in/oauth2.v3"
	oauth2_models "gopkg.in/oauth2.v3/models"
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
	out := make(chan error, 1)
	defer close(out)

	c := token.Create{
		TokenInfo: info,
	}

	go func() {
		ts.commandBus.Publish(ctx, c, out)
	}()

	ctxDoneCh := ctx.Done()
	select {
	case <-ctxDoneCh:
		return errors.Wrap(ctx.Err(), errors.TIMEOUT, "Context done")
	case err := <-out:
		if err != nil {
			return errors.Wrap(err, errors.INTERNAL, "Create token failed")
		}
		return nil
	}
}

// RemoveByCode use the authorization code to delete the token information
func (ts *TokenStore) RemoveByCode(code string) error {
	ctx := context.Background()
	t, err := ts.repository.GetByCode(ctx, code)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Token RemoveByCode error")
	}

	return ts.remove(ctx, t)
}

// RemoveByAccess use the access token to delete the token information
func (ts *TokenStore) RemoveByAccess(access string) error {
	ctx := context.Background()
	t, err := ts.repository.GetByAccess(ctx, access)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Token RemoveByAccess error")
	}

	return ts.remove(ctx, t)
}

// RemoveByRefresh use the refresh token to delete the token information
func (ts *TokenStore) RemoveByRefresh(refresh string) error {
	ctx := context.Background()
	t, err := ts.repository.GetByRefresh(ctx, refresh)
	if err != nil {
		return errors.Wrap(err, errors.INTERNAL, "Token RemoveByRefresh error")
	}

	return ts.remove(ctx, t)
}

// GetByCode use the authorization code for token information data
func (ts *TokenStore) GetByCode(code string) (oauth2.TokenInfo, error) {
	t, err := ts.repository.GetByCode(context.Background(), code)
	if err != nil {
		return nil, errors.Wrap(err, errors.INTERNAL, "Token not found")
	}

	return ts.toTokenInfo(t.GetData())
}

// GetByAccess use the access token for token information data
func (ts *TokenStore) GetByAccess(access string) (oauth2.TokenInfo, error) {
	t, err := ts.repository.GetByAccess(context.Background(), access)
	if err != nil {
		return nil, errors.Wrap(err, errors.INTERNAL, "Token not found")
	}

	return ts.toTokenInfo(t.GetData())
}

// GetByRefresh use the refresh token for token information data
func (ts *TokenStore) GetByRefresh(refresh string) (oauth2.TokenInfo, error) {
	t, err := ts.repository.GetByRefresh(context.Background(), refresh)
	if err != nil {
		return nil, errors.Wrap(err, errors.INTERNAL, "Token not found")
	}

	return ts.toTokenInfo(t.GetData())
}

func (ts *TokenStore) toTokenInfo(data []byte) (oauth2.TokenInfo, error) {
	info := oauth2_models.Token{}
	err := json.Unmarshal(data, &info)
	if err != nil {
		return nil, errors.Wrap(err, errors.INTERNAL, "Unmarshal token failed")
	}

	return &info, nil
}

func (ts *TokenStore) remove(ctx context.Context, t persistence.Token) error {
	out := make(chan error, 1)
	defer close(out)

	c := token.Remove{
		ID: uuid.MustParse(t.GetID()),
	}

	go func() {
		ts.commandBus.Publish(ctx, c, out)
	}()

	ctxDoneCh := ctx.Done()
	select {
	case <-ctxDoneCh:
		return errors.Wrap(ctx.Err(), errors.TIMEOUT, "Context done")
	case err := <-out:
		if err != nil {
			return errors.Wrap(err, errors.INTERNAL, "Token remove failed")
		}
		return nil
	}
}
