package user

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/executioncontext"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/commandbus"
)

// ChangeEmailAddress command
type ChangeEmailAddress struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

// UnmarshalJSON implements json.Unmarshaler interface
func (c *ChangeEmailAddress) UnmarshalJSON(payload []byte) error {
	return json.Unmarshal(payload, c)
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

		out <- repository.Save(executioncontext.ContextWithFlag(ctx, executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
}

// RegisterWithEmail command
type RegisterWithEmail struct {
	Email string `json:"email"`
}

// UnmarshalJSON implements json.Unmarshaler interface
func (c *RegisterWithEmail) UnmarshalJSON(payload []byte) error {
	return json.Unmarshal(payload, c)
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

		out <- repository.Save(executioncontext.ContextWithFlag(ctx, executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
}

// RegisterWithFacebook command
type RegisterWithFacebook struct {
	Email string `json:"email"`
}

// UnmarshalJSON implements json.Unmarshaler interface
func (c *RegisterWithFacebook) UnmarshalJSON(payload []byte) error {
	return json.Unmarshal(payload, c)
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
		err = u.RegisterWithFacebook(id, c.Email)
		if err != nil {
			out <- err
			return
		}

		out <- repository.Save(executioncontext.ContextWithFlag(ctx, executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
}

// RegisterWithGoogle command
type RegisterWithGoogle struct {
	Email string `json:"email"`
}

// UnmarshalJSON implements json.Unmarshaler interface
func (c *RegisterWithGoogle) UnmarshalJSON(payload []byte) error {
	return json.Unmarshal(payload, c)
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
		err = u.RegisterWithGoogle(id, c.Email)
		if err != nil {
			out <- err
			return
		}

		out <- repository.Save(executioncontext.ContextWithFlag(ctx, executioncontext.LIVE), u)
	}

	return commandbus.CommandHandler(fn)
}
