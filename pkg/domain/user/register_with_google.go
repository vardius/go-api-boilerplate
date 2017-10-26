package user

import (
	"app/pkg/domain"
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// RegisterWithGoogle command bus contract
const RegisterWithGoogle = "register_with_google"

type registerWithGoogle struct {
	Email     string `json:"email"`
	AuthToken string `json:"authToken"`
}

func (c *registerWithGoogle) fromJSON(payload json.RawMessage) error {
	return json.Unmarshal(payload, c)
}

func onRegisterWithGoogle(repository *eventSourcedRepository) domain.CommandHandler {
	return func(ctx context.Context, payload json.RawMessage, out chan<- error) {
		c := &registerWithGoogle{}
		err := c.fromJSON(payload)
		if err != nil {
			out <- err
			return
		}

		//todo: validate if email is taken or if user already connected with google

		id, err := uuid.NewRandom()
		if err != nil {
			out <- err
			return
		}

		user := New()
		err = user.RegisterWithGoogle(id, c.Email, c.AuthToken)
		if err != nil {
			out <- err
			return
		}

		out <- nil

		// todo add live flag to context
		repository.Save(ctx, user)
	}
}
