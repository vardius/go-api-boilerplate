package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
)

const (
	// ChangeUserEmailAddress command bus contract
	ChangeUserEmailAddress = "change-user-email-address"
	// RegisterUserWithEmail command bus contract
	RegisterUserWithEmail = "register-user-with-email"
	// RegisterUserWithFacebook command bus contract
	RegisterUserWithFacebook = "register-user-with-facebook"
	// RegisterUserWithGoogle command bus contract
	RegisterUserWithGoogle = "register-user-with-google"
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
func OnChangeEmailAddress(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, command domain.Command) error {
		c, ok := command.(ChangeEmailAddress)
		if !ok {
			return apperrors.New("invalid command")
		}

		var totalUsers int32

		row := db.QueryRowContext(ctx, `SELECT COUNT(distinct_id) FROM users WHERE email_address=?`, c.Email.String())
		if err := row.Scan(&totalUsers); err != nil {
			return apperrors.Wrap(err)
		}

		if totalUsers != 0 {
			return apperrors.Wrap(application.ErrInvalid)
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
func OnRegisterWithEmail(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, command domain.Command) error {
		c, ok := command.(RegisterWithEmail)
		if !ok {
			return apperrors.New("invalid command")
		}

		var userID string
		row := db.QueryRowContext(ctx, `SELECT id FROM users WHERE email_address=?`, c.Email.String())
		if err := row.Scan(&userID); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return apperrors.Wrap(err)
		}

		var u User
		if userID != "" {
			id, err := uuid.Parse(userID)
			if err != nil {
				return apperrors.Wrap(err)
			}

			u, err = repository.Get(ctx, id)
			if err != nil {
				return apperrors.Wrap(err)
			}

			if err := u.RequestAccessToken(ctx); err != nil {
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
	Email       EmailAddress `json:"email"`
	FacebookID  string       `json:"facebook_id"`
	AccessToken string       `json:"access_token"`
}

// GetName returns command name
func (c RegisterWithFacebook) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnRegisterWithFacebook creates command handler
func OnRegisterWithFacebook(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, command domain.Command) error {
		c, ok := command.(RegisterWithFacebook)
		if !ok {
			return apperrors.New("invalid command")
		}

		var id, emailAddress, facebookID string

		row := db.QueryRowContext(ctx, `SELECT id, email_address, facebook_id FROM users WHERE email_address=? OR facebook_id=? LIMIT 1`, c.Email.String(), c.FacebookID)
		if err := row.Scan(&id, &emailAddress, &facebookID); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return apperrors.Wrap(err)
		}

		if facebookID == c.FacebookID {
			return apperrors.Wrap(application.ErrInvalid)
		}

		var u User
		if emailAddress == string(c.Email) {
			userID, err := uuid.Parse(id)
			if err != nil {
				return apperrors.Wrap(err)
			}

			u, err := repository.Get(ctx, userID)
			if err != nil {
				return apperrors.Wrap(err)
			}

			if err := u.ConnectWithFacebook(ctx, c.FacebookID, c.AccessToken); err != nil {
				return apperrors.Wrap(err)
			}
		} else {
			id, err := uuid.NewRandom()
			if err != nil {
				return apperrors.Wrap(err)
			}

			u = New()

			if err := u.RegisterWithFacebook(ctx, id, c.Email, c.FacebookID, c.AccessToken); err != nil {
				return apperrors.Wrap(err)
			}
		}

		if err := repository.SaveAndAcknowledge(executioncontext.WithFlag(ctx, executioncontext.LIVE), u); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return fn
}

// RegisterWithGoogle command
type RegisterWithGoogle struct {
	Email       EmailAddress `json:"email"`
	GoogleID    string       `json:"google_id"`
	AccessToken string       `json:"access_token"`
}

// GetName returns command name
func (c RegisterWithGoogle) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnRegisterWithGoogle creates command handler
func OnRegisterWithGoogle(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, command domain.Command) error {
		c, ok := command.(RegisterWithGoogle)
		if !ok {
			return apperrors.New("invalid command")
		}

		var id, emailAddress, googleID string

		row := db.QueryRowContext(ctx, `SELECT id, email_address, google_id FROM users WHERE email_address=? OR google_id=? LIMIT 1`, c.Email.String(), c.GoogleID)
		if err := row.Scan(&id, &emailAddress, &googleID); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return apperrors.Wrap(err)
		}

		if googleID == c.GoogleID {
			return apperrors.Wrap(application.ErrInvalid)
		}

		var u User
		if emailAddress == string(c.Email) {
			userID, err := uuid.Parse(id)
			if err != nil {
				return apperrors.Wrap(err)
			}

			u, err := repository.Get(ctx, userID)
			if err != nil {
				return apperrors.Wrap(err)
			}

			if err := u.ConnectWithGoogle(ctx, c.GoogleID, c.AccessToken); err != nil {
				return apperrors.Wrap(err)
			}
		} else {
			id, err := uuid.NewRandom()
			if err != nil {
				return apperrors.Wrap(err)
			}

			u = New()
			if err := u.RegisterWithGoogle(ctx, id, c.Email, c.GoogleID, c.AccessToken); err != nil {
				return apperrors.Wrap(err)
			}
		}

		if err := repository.SaveAndAcknowledge(executioncontext.WithFlag(ctx, executioncontext.LIVE), u); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return fn
}
