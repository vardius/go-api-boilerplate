package token

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gopkg.in/oauth2.v4/models"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/access"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
	"github.com/vardius/go-api-boilerplate/pkg/metadata"
)

const (
	// CreateAuthToken command bus contract
	CreateAuthToken = "token-create"
	// RemoveAuthToken command bus contract
	RemoveAuthToken = "token-remove"
)

var (
	RemoveName = (Remove{}).GetName()
	CreateName = (Create{}).GetName()
)

// NewCommandFromPayload builds command by contract from json payload
func NewCommandFromPayload(contract string, payload []byte) (domain.Command, error) {
	switch contract {
	case CreateAuthToken:
		var command Create
		return command, nil
	case RemoveAuthToken:
		var command Remove
		if err := json.Unmarshal(payload, &command); err != nil {
			return command, apperrors.Wrap(err)
		}
		return command, nil
	default:
		return nil, apperrors.Wrap(fmt.Errorf("invalid command contract: %s", contract))
	}
}

// Create command, creates access token for user
type Create struct{}

// GetName returns command name
func (c Create) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnCreate creates command handler
func OnCreate(repository Repository) commandbus.CommandHandler {
	fn := func(ctx context.Context, command domain.Command) error {
		i, hasIdentity := identity.FromContext(ctx)
		if !hasIdentity {
			return apperrors.Wrap(apperrors.ErrUnauthorized)
		}

		id, err := uuid.NewRandom()
		if err != nil {
			return apperrors.Wrap(fmt.Errorf("%w: Could not generate new id: %s", apperrors.ErrInternal, err))
		}

		var userAgent string
		if m, ok := metadata.FromContext(ctx); ok {
			userAgent = m.UserAgent
		}

		token := New()
		if err := token.Create(ctx, id, uuid.Nil, i.UserID, &models.Token{
			ClientID:        uuid.Nil.String(),
			UserID:          i.UserID.String(),
			Scope:           string(access.ScopeAll),
			Access:          i.Token,
			AccessCreateAt:  time.Now(),
			AccessExpiresIn: 365 * 24 * time.Hour,
		}, userAgent); err != nil {
			return apperrors.Wrap(fmt.Errorf("%w: Error when creating token: %s", apperrors.ErrInternal, err))
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), token); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return fn
}

// Remove command
type Remove struct {
	ID uuid.UUID `json:"id"`
}

// GetName returns command name
func (c Remove) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnRemove creates command handler
func OnRemove(repository Repository) commandbus.CommandHandler {
	fn := func(ctx context.Context, command domain.Command) error {
		c, ok := command.(Remove)
		if !ok {
			return apperrors.New("invalid command")
		}

		i, hasIdentity := identity.FromContext(ctx)
		if !hasIdentity {
			return apperrors.Wrap(apperrors.ErrUnauthorized)
		}

		token, err := repository.Get(ctx, c.ID)
		if err != nil {
			return apperrors.Wrap(err)
		}
		if i.UserID.String() != token.userID.String() {
			return apperrors.Wrap(apperrors.ErrForbidden)
		}

		if err := token.Remove(ctx); err != nil {
			return apperrors.Wrap(fmt.Errorf("%w: Error when removing token: %s", apperrors.ErrInternal, err))
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), token); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return fn
}
