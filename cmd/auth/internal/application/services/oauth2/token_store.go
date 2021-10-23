package oauth2

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gopkg.in/oauth2.v4"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
	"github.com/vardius/go-api-boilerplate/pkg/metadata"
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
		return apperrors.Wrap(fmt.Errorf("%w: Could not generate new id: %s", apperrors.ErrInternal, err))
	}

	var userID uuid.UUID
	if info.GetUserID() != "" {
		userUUID, err := uuid.Parse(info.GetUserID())
		if err != nil {
			return apperrors.Wrap(err)
		}
		userID = userUUID
	}

	clientID, err := uuid.Parse(info.GetClientID())
	if err != nil {
		return apperrors.Wrap(err)
	}

	var userAgent string
	if m, ok := metadata.FromContext(ctx); ok {
		userAgent = m.UserAgent
	}

	t := token.New()
	if err := t.Create(ctx, id, clientID, userID, info, userAgent); err != nil {
		return apperrors.Wrap(err)
	}

	if err := ts.eventSourcedRepository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), t); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

// RemoveByCode use the authorization code to delete the token information
func (ts *TokenStore) RemoveByCode(ctx context.Context, code string) error {
	t, err := ts.persistenceRepository.GetByCode(ctx, code)
	if err != nil {
		return apperrors.Wrap(err)
	}

	return ts.remove(ctx, t)
}

// RemoveByAccess use the access token to delete the token information
func (ts *TokenStore) RemoveByAccess(ctx context.Context, access string) error {
	t, err := ts.persistenceRepository.GetByAccess(ctx, access)
	if err != nil {
		return apperrors.Wrap(err)
	}

	return ts.remove(ctx, t)
}

// RemoveByRefresh use the refresh token to delete the token information
func (ts *TokenStore) RemoveByRefresh(ctx context.Context, refresh string) error {
	t, err := ts.persistenceRepository.GetByRefresh(ctx, refresh)
	if err != nil {
		return apperrors.Wrap(err)
	}

	return ts.remove(ctx, t)
}

// GetByCode use the authorization code for token information data
func (ts *TokenStore) GetByCode(ctx context.Context, code string) (oauth2.TokenInfo, error) {
	t, err := ts.persistenceRepository.GetByCode(ctx, code)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	info, err := t.TokenInfo()
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	return info, nil
}

// GetByAccess use the access token for token information data
func (ts *TokenStore) GetByAccess(ctx context.Context, access string) (oauth2.TokenInfo, error) {
	t, err := ts.persistenceRepository.GetByAccess(ctx, access)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	info, err := t.TokenInfo()
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	return info, nil
}

// GetByRefresh use the refresh token for token information data
func (ts *TokenStore) GetByRefresh(ctx context.Context, refresh string) (oauth2.TokenInfo, error) {
	t, err := ts.persistenceRepository.GetByRefresh(ctx, refresh)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	info, err := t.TokenInfo()
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	return info, nil
}

func (ts *TokenStore) remove(ctx context.Context, model persistence.Token) error {
	id, err := uuid.Parse(model.GetID())
	if err != nil {
		return apperrors.Wrap(err)
	}

	t, err := ts.eventSourcedRepository.Get(ctx, id)
	if err != nil {
		return apperrors.Wrap(err)
	}

	if err := t.Remove(ctx); err != nil {
		return apperrors.Wrap(fmt.Errorf("%w: Error when removing token: %s", apperrors.ErrInternal, err))
	}

	if err := ts.eventSourcedRepository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), t); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}
