package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
)

const (
	// ChangeUserEmailAddress command bus contract
	ChangeUserEmailAddress = "user-change-email-address"
	// RequestUserAccessToken command bus contract
	RequestUserAccessToken = "user-request-access-token"
	// RegisterUserWithEmail command bus contract
	RegisterUserWithEmail = "user-register-with-email"
	// RegisterUserWithFacebook command bus contract
	RegisterUserWithFacebook = "user-register-with-facebook"
	// RegisterUserWithGoogle command bus contract
	RegisterUserWithGoogle = "user-register-with-google"
)

var (
	RegisterWithEmailName    = (RegisterWithEmail{}).GetName()
	RequestAccessTokenName   = (RequestAccessToken{}).GetName()
	RegisterWithGoogleName   = (RegisterWithGoogle{}).GetName()
	RegisterWithFacebookName = (RegisterWithFacebook{}).GetName()
	ChangeEmailAddressName   = (ChangeEmailAddress{}).GetName()
)

// NewCommandFromPayload builds command by contract from json payload
func NewCommandFromPayload(contract string, payload []byte) (domain.Command, error) {
	switch contract {
	case RegisterUserWithEmail:
		var command RegisterWithEmail
		if err := json.Unmarshal(payload, &command); err != nil {
			return command, apperrors.Wrap(err)
		}

		return command, nil
	case RegisterUserWithGoogle:
		var command RegisterWithGoogle
		if err := json.Unmarshal(payload, &command); err != nil {
			return command, apperrors.Wrap(err)
		}

		return command, nil
	case RegisterUserWithFacebook:
		var command RegisterWithFacebook
		if err := json.Unmarshal(payload, &command); err != nil {
			return command, apperrors.Wrap(err)
		}

		return command, nil
	case ChangeUserEmailAddress:
		var command ChangeEmailAddress
		if err := json.Unmarshal(payload, &command); err != nil {
			return command, apperrors.Wrap(err)
		}

		return command, nil
	case RequestUserAccessToken:
		var command RequestAccessToken
		if err := json.Unmarshal(payload, &command); err != nil {
			return command, apperrors.Wrap(err)
		}

		return command, nil
	default:
		return nil, apperrors.Wrap(fmt.Errorf("invalid command contract: %s", contract))
	}
}

// ChangeEmailAddress command
type ChangeEmailAddress struct {
	ID    uuid.UUID    `json:"id"`
	Email EmailAddress `json:"email"`
}

// GetName returns command name
func (c ChangeEmailAddress) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnChangeEmailAddress creates command handler
func OnChangeEmailAddress(repository Repository, userRepository persistence.UserRepository) commandbus.CommandHandler {
	fn := func(ctx context.Context, command domain.Command) error {
		c, ok := command.(ChangeEmailAddress)
		if !ok {
			return apperrors.New("invalid command")
		}

		if _, err := userRepository.GetByEmail(ctx, c.Email.String()); err != nil && !errors.Is(err, apperrors.ErrNotFound) {
			return apperrors.Wrap(err)
		} else if err == nil {
			return apperrors.Wrap(fmt.Errorf("%w: user with this email address is already registered", apperrors.ErrInvalid))
		}

		u, err := repository.Get(ctx, c.ID)
		if err != nil {
			return apperrors.Wrap(err)
		}

		if err := u.ChangeEmailAddress(ctx, c.Email); err != nil {
			return apperrors.Wrap(err)
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), u); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return fn
}

// RequestAccessToken command
type RequestAccessToken struct {
	ID           uuid.UUID `json:"id"`
	RedirectPath string    `json:"redirect_path,omitempty"`
}

// GetName returns command name
func (c RequestAccessToken) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnRequestAccessToken creates command handler
func OnRequestAccessToken(repository Repository) commandbus.CommandHandler {
	fn := func(ctx context.Context, command domain.Command) error {
		c, ok := command.(RequestAccessToken)
		if !ok {
			return apperrors.New("invalid command")
		}

		u, err := repository.Get(ctx, c.ID)
		if err != nil {
			return apperrors.Wrap(err)
		}

		if err := u.RequestAccessToken(ctx, c.RedirectPath); err != nil {
			return apperrors.Wrap(err)
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), u); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return fn
}

// RegisterWithEmail command
type RegisterWithEmail struct {
	Email        EmailAddress `json:"email"`
	RedirectPath string       `json:"redirect_path,omitempty"`
}

// GetName returns command name
func (c RegisterWithEmail) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnRegisterWithEmail creates command handler
func OnRegisterWithEmail(repository Repository, userRepository persistence.UserRepository) commandbus.CommandHandler {
	fn := func(ctx context.Context, command domain.Command) error {
		c, ok := command.(RegisterWithEmail)
		if !ok {
			return apperrors.New("invalid command")
		}

		user, err := userRepository.GetByEmail(ctx, c.Email.String())
		if err != nil && !errors.Is(err, apperrors.ErrNotFound) {
			return apperrors.Wrap(err)
		}

		var u User
		if err == nil {
			id, err := uuid.Parse(user.GetID())
			if err != nil {
				return apperrors.Wrap(err)
			}

			u, err = repository.Get(ctx, id)
			if err != nil {
				return apperrors.Wrap(err)
			}

			if err := u.RequestAccessToken(ctx, c.RedirectPath); err != nil {
				return apperrors.Wrap(err)
			}
		} else {
			id, err := uuid.NewRandom()
			if err != nil {
				return apperrors.Wrap(err)
			}

			u = New()
			if err := u.RegisterWithEmail(ctx, id, c.Email); err != nil {
				return apperrors.Wrap(err)
			}
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), u); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return fn
}

// RegisterWithFacebook command
type RegisterWithFacebook struct {
	Email        EmailAddress `json:"email"`
	FacebookID   string       `json:"facebook_id"`
	AccessToken  string       `json:"access_token"`
	RedirectPath string       `json:"redirect_path,omitempty"`
}

// GetName returns command name
func (c RegisterWithFacebook) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnRegisterWithFacebook creates command handler
func OnRegisterWithFacebook(repository Repository, userRepository persistence.UserRepository) commandbus.CommandHandler {
	fn := func(ctx context.Context, command domain.Command) error {
		c, ok := command.(RegisterWithFacebook)
		if !ok {
			return apperrors.New("invalid command")
		}

		var user User
		if u, err := userRepository.GetByFacebookID(ctx, c.FacebookID); err != nil && !errors.Is(err, apperrors.ErrNotFound) {
			return apperrors.Wrap(err)
		} else if err == nil {
			if u.GetEmail() != c.Email.String() {
				return apperrors.Wrap(fmt.Errorf("%w: facebook account connected to another user", apperrors.ErrInvalid))
			}

			id, err := uuid.Parse(u.GetID())
			if err != nil {
				return apperrors.Wrap(err)
			}

			user, err = repository.Get(ctx, id)
			if err != nil {
				return apperrors.Wrap(err)
			}

			if err := user.RequestAccessToken(ctx, c.RedirectPath); err != nil {
				return apperrors.Wrap(err)
			}
		} else {
			if u, err := userRepository.GetByEmail(ctx, c.Email.String()); err != nil && !errors.Is(err, apperrors.ErrNotFound) {
				return apperrors.Wrap(err)
			} else if err == nil {
				if u.GetFacebookID() != "" && u.GetFacebookID() != c.FacebookID {
					return apperrors.Wrap(fmt.Errorf("%w: user connected to another facebook account", apperrors.ErrInvalid))
				}

				userID, err := uuid.Parse(u.GetID())
				if err != nil {
					return apperrors.Wrap(err)
				}

				user, err = repository.Get(ctx, userID)
				if err != nil {
					return apperrors.Wrap(err)
				}

				if err := user.ConnectWithFacebook(ctx, c.FacebookID, c.AccessToken, c.RedirectPath); err != nil {
					return apperrors.Wrap(err)
				}
			} else {
				id, err := uuid.NewRandom()
				if err != nil {
					return apperrors.Wrap(err)
				}

				user = New()
				if err := user.RegisterWithFacebook(ctx, id, c.Email, c.FacebookID, c.AccessToken, c.RedirectPath); err != nil {
					return apperrors.Wrap(err)
				}
			}
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), user); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return fn
}

// RegisterWithGoogle command
type RegisterWithGoogle struct {
	Email        EmailAddress `json:"email"`
	GoogleID     string       `json:"google_id"`
	AccessToken  string       `json:"access_token"`
	RedirectPath string       `json:"redirect_path,omitempty"`
}

// GetName returns command name
func (c RegisterWithGoogle) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnRegisterWithGoogle creates command handler
func OnRegisterWithGoogle(repository Repository, userRepository persistence.UserRepository) commandbus.CommandHandler {
	fn := func(ctx context.Context, command domain.Command) error {
		c, ok := command.(RegisterWithGoogle)
		if !ok {
			return apperrors.New("invalid command")
		}

		var user User
		if u, err := userRepository.GetByGoogleID(ctx, c.GoogleID); err != nil && !errors.Is(err, apperrors.ErrNotFound) {
			return apperrors.Wrap(err)
		} else if err == nil {
			if u.GetEmail() != c.Email.String() {
				return apperrors.Wrap(fmt.Errorf("%w: google account connected to another user", apperrors.ErrInvalid))
			}

			id, err := uuid.Parse(u.GetID())
			if err != nil {
				return apperrors.Wrap(err)
			}

			user, err = repository.Get(ctx, id)
			if err != nil {
				return apperrors.Wrap(err)
			}

			if err := user.RequestAccessToken(ctx, c.RedirectPath); err != nil {
				return apperrors.Wrap(err)
			}
		} else {
			if u, err := userRepository.GetByEmail(ctx, c.Email.String()); err != nil && !errors.Is(err, apperrors.ErrNotFound) {
				return apperrors.Wrap(err)
			} else if err == nil {
				if u.GetGoogleID() != "" && u.GetGoogleID() != c.GoogleID {
					return apperrors.Wrap(fmt.Errorf("%w: user connected to another google account", apperrors.ErrInvalid))
				}

				userID, err := uuid.Parse(u.GetID())
				if err != nil {
					return apperrors.Wrap(err)
				}

				user, err = repository.Get(ctx, userID)
				if err != nil {
					return apperrors.Wrap(err)
				}

				if err := user.ConnectWithGoogle(ctx, c.GoogleID, c.AccessToken, c.RedirectPath); err != nil {
					return apperrors.Wrap(err)
				}
			} else {
				id, err := uuid.NewRandom()
				if err != nil {
					return apperrors.Wrap(err)
				}

				user = New()
				if err := user.RegisterWithGoogle(ctx, id, c.Email, c.GoogleID, c.AccessToken, c.RedirectPath); err != nil {
					return apperrors.Wrap(err)
				}
			}
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), user); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return fn
}
