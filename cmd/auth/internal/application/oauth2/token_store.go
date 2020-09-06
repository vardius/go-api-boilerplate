package oauth2

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gopkg.in/oauth2.v4"
	oauth2models "gopkg.in/oauth2.v4/models"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
)

// NewTokenStore create a token store instance
func NewTokenStore(persistenceRepository persistence.TokenRepository, eventSourcedRepository token.Repository) *TokenStore {
	return &TokenStore{
		persistenceRepository:  persistenceRepository,
		eventSourcedRepository: eventSourcedRepository,
	}
}

// TokenStore token storage
type TokenStore struct {
	persistenceRepository  persistence.TokenRepository
	eventSourcedRepository token.Repository
}

// Create create and store the new token information
func (ts *TokenStore) Create(ctx context.Context, info oauth2.TokenInfo) error {
	id, err := uuid.NewRandom()
	if err != nil {
		return errors.Wrap(fmt.Errorf("%w: Could not generate new id: %s", application.ErrInternal, err))
	}

	userID, err := uuid.Parse(info.GetUserID())
	if err != nil {
		return errors.Wrap(err)
	}

	clientID, err := uuid.Parse(info.GetClientID())
	if err != nil {
		return errors.Wrap(err)
	}

	t := token.New()
	if err := t.Create(ctx, id, clientID, userID, info.GetCode(), info.GetScope(), info.GetAccess(), info.GetRefresh()); err != nil {
		return errors.Wrap(err)
	}

	if err := ts.eventSourcedRepository.SaveAndAcknowledge(executioncontext.WithFlag(ctx, executioncontext.LIVE), t); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

// RemoveByCode use the authorization code to delete the token information
func (ts *TokenStore) RemoveByCode(ctx context.Context, code string) error {
	t, err := ts.persistenceRepository.GetByCode(ctx, code)
	if err != nil {
		return errors.Wrap(err)
	}

	return ts.remove(ctx, t)
}

// RemoveByAccess use the access token to delete the token information
func (ts *TokenStore) RemoveByAccess(ctx context.Context, access string) error {
	t, err := ts.persistenceRepository.GetByAccess(ctx, access)
	if err != nil {
		return errors.Wrap(err)
	}

	return ts.remove(ctx, t)
}

// RemoveByRefresh use the refresh token to delete the token information
func (ts *TokenStore) RemoveByRefresh(ctx context.Context, refresh string) error {
	t, err := ts.persistenceRepository.GetByRefresh(ctx, refresh)
	if err != nil {
		return errors.Wrap(err)
	}

	return ts.remove(ctx, t)
}

// GetByCode use the authorization code for token information data
func (ts *TokenStore) GetByCode(ctx context.Context, code string) (oauth2.TokenInfo, error) {
	t, err := ts.persistenceRepository.GetByCode(ctx, code)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	info, err := ts.toTokenInfo(t)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return info, nil
}

// GetByAccess use the access token for token information data
func (ts *TokenStore) GetByAccess(ctx context.Context, access string) (oauth2.TokenInfo, error) {
	t, err := ts.persistenceRepository.GetByAccess(ctx, access)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	info, err := ts.toTokenInfo(t)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return info, nil
}

// GetByRefresh use the refresh token for token information data
func (ts *TokenStore) GetByRefresh(ctx context.Context, refresh string) (oauth2.TokenInfo, error) {
	t, err := ts.persistenceRepository.GetByRefresh(ctx, refresh)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	info, err := ts.toTokenInfo(t)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return info, nil
}

func (ts *TokenStore) toTokenInfo(token persistence.Token) (oauth2.TokenInfo, error) {
	info := oauth2models.Token{
		ClientID: token.GetClientID(),
		UserID:   token.GetUserID(),
		Access:   token.GetAccess(),
		Refresh:  token.GetRefresh(),
		Scope:    token.GetScope(),
		Code:     token.GetCode(),
	}

	return &info, nil
}

func (ts *TokenStore) remove(ctx context.Context, model persistence.Token) error {
	id, err := uuid.Parse(model.GetID())
	if err != nil {
		return errors.Wrap(err)
	}

	t, err := ts.eventSourcedRepository.Get(ctx, id)
	if err != nil {
		return errors.Wrap(err)
	}

	if err := t.Remove(ctx); err != nil {
		return errors.Wrap(fmt.Errorf("%w: Error when removing token: %s", application.ErrInternal, err))
	}

	if err := ts.eventSourcedRepository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), t); err != nil {
		return errors.Wrap(err)
	}

	return nil
}
