package identity

import (
	"context"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

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
