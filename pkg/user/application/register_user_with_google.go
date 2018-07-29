package application

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/executioncontext"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventstore"
	"github.com/vardius/go-api-boilerplate/pkg/user/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/user/infrastructure"
)

type registerUserWithGoogle struct {
	Email     string `json:"email"`
	AuthToken string `json:"authToken"`
}

func (c *registerUserWithGoogle) fromJSON(payload json.RawMessage) error {
	return json.Unmarshal(payload, c)
}

// OnRegisterUserWithGoogle creates command handler
func OnRegisterUserWithGoogle(es eventstore.EventStore, eb eventbus.EventBus) commandbus.CommandHandler {
	repository := infrastructure.NewUserEventSourcedRepository(fmt.Sprintf("%T", user.User{}), es, eb)

	return func(ctx context.Context, payload json.RawMessage, out chan<- error) {
		c := &registerUserWithGoogle{}
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

		u := user.New()
		err = u.RegisterWithGoogle(id, c.Email, c.AuthToken)
		if err != nil {
			out <- err
			return
		}

		out <- repository.Save(executioncontext.ContextWithFlag(ctx, executioncontext.LIVE), u)
	}
}
