package user

import (
	"fmt"
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

// RegisterWithFacebook command bus contract
const RegisterWithFacebook = "register-user-with-facebook"

type registerWithFacebook struct {
	Email     string `json:"email"`
	AuthToken string `json:"authToken"`
}

func (c *registerWithFacebook) fromJSON(payload json.RawMessage) error {
	return json.Unmarshal(payload, c)
}

// OnRegisterWithFacebook creates command handler
func OnRegisterWithFacebook(es domain.EventStore, eb domain.EventBus) domain.CommandHandler {
	repository := newRepository(fmt.Sprintf("%T", User{}), es, eb)

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

		repository.Save(domain.ContextWithFlag(ctx, domain.LIVE), user)
	}
}
