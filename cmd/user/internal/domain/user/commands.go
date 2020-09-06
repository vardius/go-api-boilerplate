package user

import (
	"context"
	"database/sql"
	systemErrors "errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
)

const (
	// RequestUserAccessToken command bus contract
	RequestUserAccessToken = "request-user-access-token"
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
		registerWithEmail := RegisterWithEmail{}
		err := unmarshalPayload(payload, &registerWithEmail)

		return registerWithEmail, errors.Wrap(err)
	case RegisterUserWithGoogle:
		registerWithGoogle := RegisterWithGoogle{}
		err := unmarshalPayload(payload, &registerWithGoogle)

		return registerWithGoogle, errors.Wrap(err)
	case RegisterUserWithFacebook:
		registerWithFacebook := RegisterWithFacebook{}
		err := unmarshalPayload(payload, &registerWithFacebook)

		return registerWithFacebook, errors.Wrap(err)
	case ChangeUserEmailAddress:
		changeEmailAddress := ChangeEmailAddress{}
		err := unmarshalPayload(payload, &changeEmailAddress)

		return changeEmailAddress, errors.Wrap(err)
	case RequestUserAccessToken:
		requestAccessToken := RequestAccessToken{}
		err := unmarshalPayload(payload, &requestAccessToken)

		return requestAccessToken, errors.Wrap(err)
	default:
		return nil, errors.New("Invalid command contract")
	}
}

// RequestAccessToken command
type RequestAccessToken struct {
	Email        EmailAddress `json:"email"`
	RedirectPath string       `json:"redirect_path,omitempty"`
}

// GetName returns command name
func (c RequestAccessToken) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnRequestAccessToken creates command handler
func OnRequestAccessToken(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, command domain.Command) error {
		c, ok := command.(RequestAccessToken)
		if !ok {
			return errors.New("invalid command")
		}

		var id string
		row := db.QueryRowContext(ctx, `SELECT id FROM users WHERE email_address=? LIMIT 1`, c.Email.String())
		if err := row.Scan(&id); err != nil {
			if systemErrors.Is(err, sql.ErrNoRows) {
				return errors.Wrap(fmt.Errorf("%s: %w", err, application.ErrNotFound))
			}
			return errors.Wrap(err)
		}
		if id == "" {
			return application.ErrNotFound
		}

		userID, err := uuid.Parse(id)
		if err != nil {
			return errors.Wrap(err)
		}

		u, err := repository.Get(ctx, userID)
		if err != nil {
			return errors.Wrap(err)
		}

		if err := u.RequestAccessToken(ctx); err != nil {
			return errors.Wrap(err)
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), u); err != nil {
			return errors.Wrap(err)
		}

		return nil
	}

	return fn
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
			return errors.New("invalid command")
		}

		var totalUsers int32

		row := db.QueryRowContext(ctx, `SELECT COUNT(distinct_id) FROM users WHERE email_address=?`, c.Email.String())
		if err := row.Scan(&totalUsers); err != nil {
			return errors.Wrap(err)
		}

		if totalUsers != 0 {
			return errors.Wrap(application.ErrInvalid)
		}

		u, err := repository.Get(ctx, c.ID)
		if err != nil {
			return errors.Wrap(err)
		}

		if err := u.ChangeEmailAddress(ctx, c.Email); err != nil {
			return errors.Wrap(err)
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), u); err != nil {
			return errors.Wrap(err)
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
			return errors.New("invalid command")
		}

		var totalUsers int32

		row := db.QueryRowContext(ctx, `SELECT COUNT(distinct_id) FROM users WHERE email_address=?`, c.Email.String())
		if err := row.Scan(&totalUsers); err != nil {
			return errors.Wrap(err)
		}

		if totalUsers != 0 {
			return errors.Wrap(application.ErrInvalid)
		}

		id, err := uuid.NewRandom()
		if err != nil {
			return errors.Wrap(err)
		}

		u := New()
		if err := u.RegisterWithEmail(ctx, id, c.Email); err != nil {
			return errors.Wrap(err)
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), u); err != nil {
			return errors.Wrap(err)
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
			return errors.New("invalid command")
		}

		var id, emailAddress, facebookID string

		row := db.QueryRowContext(ctx, `SELECT id, email_address, facebook_id FROM users WHERE email_address=? OR facebook_id=? LIMIT 1`, c.Email.String(), c.FacebookID)
		if err := row.Scan(&id, &emailAddress, &facebookID); err != nil && !systemErrors.Is(err, sql.ErrNoRows) {
			return errors.Wrap(err)
		}

		if facebookID == c.FacebookID {
			return errors.Wrap(application.ErrInvalid)
		}

		var u User
		if emailAddress == string(c.Email) {
			userID, err := uuid.Parse(id)
			if err != nil {
				return errors.Wrap(err)
			}

			u, err := repository.Get(ctx, userID)
			if err != nil {
				return errors.Wrap(err)
			}

			if err := u.ConnectWithFacebook(ctx, c.FacebookID, c.AccessToken); err != nil {
				return errors.Wrap(err)
			}
		} else {
			id, err := uuid.NewRandom()
			if err != nil {
				return errors.Wrap(err)
			}

			u = New()

			if err := u.RegisterWithFacebook(ctx, id, c.Email, c.FacebookID, c.AccessToken); err != nil {
				return errors.Wrap(err)
			}
		}

		if err := repository.SaveAndAcknowledge(executioncontext.WithFlag(ctx, executioncontext.LIVE), u); err != nil {
			return errors.Wrap(err)
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
			return errors.New("invalid command")
		}

		var id, emailAddress, googleID string

		row := db.QueryRowContext(ctx, `SELECT id, email_address, google_id FROM users WHERE email_address=? OR google_id=? LIMIT 1`, c.Email.String(), c.GoogleID)
		if err := row.Scan(&id, &emailAddress, &googleID); err != nil && !systemErrors.Is(err, sql.ErrNoRows) {
			return errors.Wrap(err)
		}

		if googleID == c.GoogleID {
			return errors.Wrap(application.ErrInvalid)
		}

		var u User
		if emailAddress == string(c.Email) {
			userID, err := uuid.Parse(id)
			if err != nil {
				return errors.Wrap(err)
			}

			u, err := repository.Get(ctx, userID)
			if err != nil {
				return errors.Wrap(err)
			}

			if err := u.ConnectWithGoogle(ctx, c.GoogleID, c.AccessToken); err != nil {
				return errors.Wrap(err)
			}
		} else {
			id, err := uuid.NewRandom()
			if err != nil {
				return errors.Wrap(err)
			}

			u = New()
			if err := u.RegisterWithGoogle(ctx, id, c.Email, c.GoogleID, c.AccessToken); err != nil {
				return errors.Wrap(err)
			}
		}

		if err := repository.SaveAndAcknowledge(executioncontext.WithFlag(ctx, executioncontext.LIVE), u); err != nil {
			return errors.Wrap(err)
		}

		return nil
	}

	return fn
}
