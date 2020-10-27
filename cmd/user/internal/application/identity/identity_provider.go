package identity

import (
	"context"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

type Provider interface {
	GetByUserEmail(ctx context.Context, userEmail string) (*identity.Identity, error)
}

type identityProvider struct {
	clientRepository persistence.ClientRepository
	userRepository   persistence.UserRepository
}

func NewIdentityProvider(clientRepository persistence.ClientRepository, userRepository persistence.UserRepository) *identityProvider {
	return &identityProvider{
		clientRepository: clientRepository,
		userRepository:   userRepository,
	}
}

func (p *identityProvider) GetByUserEmail(ctx context.Context, userEmail string) (*identity.Identity, error) {
	u, err := p.userRepository.GetByEmail(ctx, userEmail)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	// We can do that because its our internal client, should have one entry per user
	c, err := p.clientRepository.GetByUserDomain(ctx, u.GetID(), config.Env.App.Domain)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	clientID, err := uuid.Parse(c.GetID())
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	clientSecret, err := uuid.Parse(c.GetSecret())
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	userID, err := uuid.Parse(u.GetID())
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	return &identity.Identity{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		ClientDomain: c.GetDomain(),
		UserID:       userID,
		UserEmail:    userEmail,
	}, nil
}

func (p *identityProvider) GetByUserID(ctx context.Context, userID, clientID uuid.UUID) (*identity.Identity, error) {
	c, err := p.clientRepository.Get(ctx, clientID.String())
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	u, err := p.userRepository.Get(ctx, userID.String())
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	clientSecret, err := uuid.Parse(c.GetSecret())
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	return &identity.Identity{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		ClientDomain: c.GetDomain(),
		UserID:       userID,
		UserEmail:    u.GetEmail(),
	}, nil
}
