package user

import (
	"app/pkg/auth"
	"app/pkg/domain"
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

const RegisterWithEmail = "register_with_email"

type registerWithEmail struct {
	Email string `json:"email"`
}

func (c *registerWithEmail) fromJSON(payload json.RawMessage) error {
	return json.Unmarshal(payload, c)
}

func onRegisterWithEmail(repository *eventSourcedRepository, jwtService auth.JwtService) domain.CommandHandler {
	return func(ctx context.Context, payload json.RawMessage, out chan<- error) {
		c := &registerWithEmail{}
		err := c.fromJSON(payload)
		if err != nil {
			out <- err
			return
		}

		//todo: validate if email is taken

		id, err := uuid.NewRandom()
		if err != nil {
			out <- err
			return
		}

		identity := auth.NewUserIdentity(id, c.Email)
		token, err := jwtService.GenerateToken(identity)
		if err != nil {
			out <- err
			return
		}

		user := New()
		err = user.RegisterWithEmail(id, c.Email, token)
		if err != nil {
			out <- err
			return
		}

		out <- nil

		// todo add live flag to context
		repository.Save(ctx, user)
	}
}
