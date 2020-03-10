package user

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"runtime/debug"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/internal/commandbus"
	"github.com/vardius/go-api-boilerplate/internal/domain"
	"github.com/vardius/go-api-boilerplate/internal/errors"
	"github.com/vardius/go-api-boilerplate/internal/executioncontext"
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

		return registerWithEmail, err
	case RegisterUserWithGoogle:
		registerWithGoogle := RegisterWithGoogle{}
		err := unmarshalPayload(payload, &registerWithGoogle)

		return registerWithGoogle, err
	case RegisterUserWithFacebook:
		registerWithFacebook := RegisterWithFacebook{}
		err := unmarshalPayload(payload, &registerWithFacebook)

		return registerWithFacebook, err
	case ChangeUserEmailAddress:
		changeEmailAddress := ChangeEmailAddress{}
		err := unmarshalPayload(payload, &changeEmailAddress)

		return changeEmailAddress, err
	case RequestUserAccessToken:
		requestAccessToken := RequestAccessToken{}
		err := unmarshalPayload(payload, &requestAccessToken)

		return requestAccessToken, err
	default:
		return nil, errors.New(errors.INTERNAL, "Invalid command contract")
	}
}

// RequestAccessToken command
type RequestAccessToken struct {
	ID uuid.UUID `json:"id"`
}

// GetName returns command name
func (c RequestAccessToken) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnRequestAccessToken creates command handler
func OnRequestAccessToken(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, c RequestAccessToken, out chan<- error) {
		// this goroutine runs independently to request's goroutine,
		// therefor recover middlewears will not recover from panic to prevent crash
		defer recoverCommandHandler(out)

		u := repository.Get(c.ID)
		err := u.RequestAccessToken()
		if err != nil {
			out <- errors.Wrap(err, errors.INTERNAL, "Error when requesting access token")
			return
		}

		out <- repository.Save(executioncontext.WithFlag(context.Background(), executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
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
	fn := func(ctx context.Context, c ChangeEmailAddress, out chan<- error) {
		// this goroutine runs independently to request's goroutine,
		// therefor recover middleware will not recover from panic to prevent crash
		defer recoverCommandHandler(out)

		var totalUsers int32

		row := db.QueryRowContext(ctx, `SELECT COUNT(distinctId) FROM users WHERE emailAddress = ?`, c.Email)
		err := row.Scan(&totalUsers)
		if err != nil {
			out <- errors.Wrap(err, errors.INTERNAL, "Could not ensure that email is not taken")
			return
		}

		if totalUsers != 0 {
			out <- errors.Wrap(err, errors.INVALID, "User with given email already registered")
			return
		}

		u := repository.Get(c.ID)
		err = u.ChangeEmailAddress(c.Email)
		if err != nil {
			out <- errors.Wrap(err, errors.INTERNAL, "Error when changing email address")
			return
		}

		out <- repository.Save(executioncontext.WithFlag(context.Background(), executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
}

// RegisterWithEmail command
type RegisterWithEmail struct {
	Email EmailAddress `json:"email"`
}

// GetName returns command name
func (c RegisterWithEmail) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnRegisterWithEmail creates command handler
func OnRegisterWithEmail(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, c RegisterWithEmail, out chan<- error) {
		// this goroutine runs independently to request's goroutine,
		// therefor recover middlewears will not recover from panic to prevent crash
		defer recoverCommandHandler(out)

		var totalUsers int32

		row := db.QueryRowContext(ctx, `SELECT COUNT(distinctId) FROM users WHERE emailAddress = ?`, c.Email)
		err := row.Scan(&totalUsers)
		if err != nil {
			out <- errors.Wrap(err, errors.INTERNAL, "Could not ensure that email is not taken")
			return
		}

		if totalUsers != 0 {
			out <- errors.Wrap(err, errors.INVALID, "User with given email already registered")
			return
		}

		id, err := uuid.NewRandom()
		if err != nil {
			out <- errors.Wrap(err, errors.INTERNAL, "Could not generate new id")
			return
		}

		u := New()
		err = u.RegisterWithEmail(id, c.Email)
		if err != nil {
			out <- errors.Wrap(err, errors.INTERNAL, "Error when registering new user")
			return
		}

		out <- repository.Save(executioncontext.WithFlag(context.Background(), executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
}

// RegisterWithFacebook command
type RegisterWithFacebook struct {
	Email      EmailAddress `json:"email"`
	FacebookID string       `json:"facebookId"`
}

// GetName returns command name
func (c RegisterWithFacebook) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnRegisterWithFacebook creates command handler
func OnRegisterWithFacebook(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, c RegisterWithFacebook, out chan<- error) {
		// this goroutine runs independently to request's goroutine,
		// therefor recover middlewears will not recover from panic to prevent crash
		defer recoverCommandHandler(out)

		var id, emailAddress, facebookID string

		row := db.QueryRowContext(ctx, `SELECT id, emailAddress, facebookId FROM users WHERE emailAddress = ? OR facebookId = ?`, c.Email, c.FacebookID)
		err := row.Scan(&id, &emailAddress, &facebookID)
		if err != nil {
			out <- errors.Wrap(err, errors.INTERNAL, "Could not ensure that user is not already registered")
			return
		}

		if facebookID == c.FacebookID {
			out <- errors.Wrap(err, errors.INVALID, "User facebook account already connected")
			return
		}

		var u User
		if emailAddress == string(c.Email) {
			u = repository.Get(uuid.MustParse(id))
			err = u.ConnectWithFacebook(c.FacebookID)
			if err != nil {
				out <- errors.Wrap(err, errors.INTERNAL, "Error when trying to connect facebook account")
				return
			}
		} else {
			id, err := uuid.NewRandom()
			if err != nil {
				out <- errors.Wrap(err, errors.INTERNAL, "Could not generate new id")
				return
			}

			u = New()
			err = u.RegisterWithFacebook(id, c.Email, c.FacebookID)
			if err != nil {
				out <- errors.Wrap(err, errors.INTERNAL, "Error when registering new user")
				return
			}
		}

		out <- repository.Save(executioncontext.WithFlag(context.Background(), executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
}

// RegisterWithGoogle command
type RegisterWithGoogle struct {
	Email    EmailAddress `json:"email"`
	GoogleID string       `json:"googleId"`
}

// GetName returns command name
func (c RegisterWithGoogle) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnRegisterWithGoogle creates command handler
func OnRegisterWithGoogle(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, c RegisterWithGoogle, out chan<- error) {
		// this goroutine runs independently to request's goroutine,
		// therefor recover middlewears will not recover from panic to prevent crash
		defer recoverCommandHandler(out)

		var id, emailAddress, googleID string

		row := db.QueryRowContext(ctx, `SELECT id, emailAddress, googleId FROM users WHERE emailAddress = ? OR googleId = ?`, c.Email, c.GoogleID)
		err := row.Scan(&id, &emailAddress, &googleID)
		if err != nil {
			out <- errors.Wrap(err, errors.INTERNAL, "Could not ensure that user is not already registered")
			return
		}

		if googleID == c.GoogleID {
			out <- errors.Wrap(err, errors.INVALID, "User google account already connected")
			return
		}

		var u User
		if emailAddress == string(c.Email) {
			u = repository.Get(uuid.MustParse(id))
			err = u.ConnectWithGoogle(c.GoogleID)
			if err != nil {
				out <- errors.Wrap(err, errors.INTERNAL, "Error when trying to connect google account")
				return
			}
		} else {
			id, err := uuid.NewRandom()
			if err != nil {
				out <- errors.Wrap(err, errors.INTERNAL, "Could not generate new id")
				return
			}

			u = New()
			err = u.RegisterWithGoogle(id, c.Email, c.GoogleID)
			if err != nil {
				out <- errors.Wrap(err, errors.INTERNAL, "Error when registering new user")
				return
			}
		}

		out <- repository.Save(executioncontext.WithFlag(context.Background(), executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
}

func recoverCommandHandler(out chan<- error) {
	if r := recover(); r != nil {
		out <- errors.Newf(errors.INTERNAL, "[CommandHandler] Recovered in %v", r)

		// Log the Go stack trace for this panic'd goroutine.
		log.Printf("%s\n", debug.Stack())
	}
}
