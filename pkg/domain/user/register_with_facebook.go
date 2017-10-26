package user

import (
	"app/pkg/domain"
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// RegisterWithFacebook command bus contract
const RegisterWithFacebook = "register_with_facebook"

type registerWithFacebook struct {
	Email     string `json:"email"`
	AuthToken string `json:"authToken"`
}

func (c *registerWithFacebook) fromJSON(payload json.RawMessage) error {
	return json.Unmarshal(payload, c)
}

func onRegisterWithFacebook(repository *eventSourcedRepository) domain.CommandHandler {
	return func(ctx context.Context, payload json.RawMessage, out chan<- error) {
		c := &registerWithFacebook{}
		err := c.fromJSON(payload)
		if err != nil {
			out <- err
			return
		}

		//todo: validate if email is taken or if user already connected with facebook

		id, err := uuid.NewRandom()
		if err != nil {
			out <- err
			return
		}

		user := New()
		err = user.RegisterWithFacebook(id, c.Email, c.AuthToken)
		if err != nil {
			out <- err
			return
		}

		out <- nil

		// todo add live flag to context
		repository.Save(ctx, user)
	}
}
