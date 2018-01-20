package user

import (
	"fmt"
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/security/identity"
)

// RegisterWithEmail command bus contract
const RegisterWithEmail = "register-user-with-email"

type registerWithEmail struct {
	Email string `json:"email"`
}

func (c *registerWithEmail) fromJSON(payload json.RawMessage) error {
	return json.Unmarshal(payload, c)
}

// OnRegisterWithEmail creates command handler
func OnRegisterWithEmail(es domain.EventStore, eb domain.EventBus, j jwt.Jwt) domain.CommandHandler {
	repository := newRepository(fmt.Sprintf("%T", User{}), es, eb)

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
