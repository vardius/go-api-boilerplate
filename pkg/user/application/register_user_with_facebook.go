package application

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/common/domain"
	"github.com/vardius/go-api-boilerplate/pkg/user/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/user/infrastructure"
)

type registerUserWithFacebook struct {
	Email     string `json:"email"`
	AuthToken string `json:"authToken"`
}

func (c *registerUserWithFacebook) fromJSON(payload json.RawMessage) error {
	return json.Unmarshal(payload, c)
}

// OnRegisterUserWithFacebook creates command handler
func OnRegisterUserWithFacebook(es domain.EventStore, eb domain.EventBus) domain.CommandHandler {
	repository := infrastructure.New(fmt.Sprintf("%T", user.User{}), es, eb)

	return func(ctx context.Context, payload json.RawMessage, out chan<- error) {
		c := &registerUserWithFacebook{}
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

		u := user.New()
		err = u.RegisterWithFacebook(id, c.Email, c.AuthToken)
		if err != nil {
			out <- err
			return
		}

		out <- nil

		repository.Save(domain.ContextWithFlag(ctx, domain.LIVE), u)
	}
}
