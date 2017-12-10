package user

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/auth/identity"
	"github.com/vardius/go-api-boilerplate/pkg/auth/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

// RegisterWithEmail command bus contract
const RegisterWithEmail = "register-user-with-email"

type registerWithEmail struct {
	Email string `json:"email"`
}

func (c *registerWithEmail) fromJSON(payload json.RawMessage) error {
	return json.Unmarshal(payload, c)
}

func onRegisterWithEmail(repository *eventSourcedRepository, j jwt.Jwt) domain.CommandHandler {
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

		i := identity.WithValues(id, c.Email, nil)
		token, err := j.Encode(i)
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

		repository.Save(domain.ContextWithFlag(ctx, domain.LIVE), user)
	}
}
