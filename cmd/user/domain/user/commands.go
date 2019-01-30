package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
)

// ChangeEmailAddress command
type ChangeEmailAddress struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

// OnChangeEmailAddress creates command handler
func OnChangeEmailAddress(repository Repository) commandbus.CommandHandler {
	fn := func(ctx context.Context, c *ChangeEmailAddress, out chan<- error) {
		//todo: validate if email is taken

		u := repository.Get(c.ID)
		err := u.ChangeEmailAddress(c.Email)
		if err != nil {
			out <- err
			return
		}

		out <- repository.Save(executioncontext.ContextWithFlag(context.Background(), executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
}

// RegisterWithEmail command
type RegisterWithEmail struct {
	Email string `json:"email"`
}

// OnRegisterWithEmail creates command handler
func OnRegisterWithEmail(repository Repository) commandbus.CommandHandler {
	fn := func(ctx context.Context, c *RegisterWithEmail, out chan<- error) {
		//todo: validate if email is taken

		id, err := uuid.NewRandom()
		if err != nil {
			out <- err
			return
		}

		u := New()
		err = u.RegisterWithEmail(id, c.Email)
		if err != nil {
			out <- err
			return
		}

		out <- repository.Save(executioncontext.ContextWithFlag(context.Background(), executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
}

// RegisterWithFacebook command
type RegisterWithFacebook struct {
	Email      string `json:"email"`
	FacebookID string `json:"facebookId"`
}

// OnRegisterWithFacebook creates command handler
func OnRegisterWithFacebook(repository Repository) commandbus.CommandHandler {
	fn := func(ctx context.Context, c *RegisterWithFacebook, out chan<- error) {
		//todo: validate if email is taken or if user already connected with facebook

		id, err := uuid.NewRandom()
		if err != nil {
			out <- err
			return
		}

		u := New()
		err = u.RegisterWithFacebook(id, c.Email, c.FacebookID)
		if err != nil {
			out <- err
			return
		}

		out <- repository.Save(executioncontext.ContextWithFlag(context.Background(), executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
}

// RegisterWithGoogle command
type RegisterWithGoogle struct {
	Email    string `json:"email"`
	GoogleID string `json:"googleId"`
}

// OnRegisterWithGoogle creates command handler
func OnRegisterWithGoogle(repository Repository) commandbus.CommandHandler {
	fn := func(ctx context.Context, c *RegisterWithGoogle, out chan<- error) {
		//todo: validate if email is taken or if user already connected with google

		id, err := uuid.NewRandom()
		if err != nil {
			out <- err
			return
		}

		u := New()
		err = u.RegisterWithGoogle(id, c.Email, c.GoogleID)
		if err != nil {
			out <- err
			return
		}

		out <- repository.Save(executioncontext.ContextWithFlag(context.Background(), executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
}
