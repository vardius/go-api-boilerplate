package application

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/executioncontext"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/security/identity"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventstore"
	"github.com/vardius/go-api-boilerplate/pkg/user/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/user/infrastructure"
)

type registerUserWithEmail struct {
	Email string `json:"email"`
}

func (c *registerUserWithEmail) fromJSON(payload json.RawMessage) error {
	return json.Unmarshal(payload, c)
}

// OnRegisterUserWithEmail creates command handler
func OnRegisterUserWithEmail(es eventstore.EventStore, eb eventbus.EventBus, j jwt.Jwt) commandbus.CommandHandler {
	repository := infrastructure.NewUserEventSourcedRepository(fmt.Sprintf("%T", user.User{}), es, eb)

	return func(ctx context.Context, payload json.RawMessage, out chan<- error) {
		c := &registerUserWithEmail{}
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

		u := user.New()
		err = u.RegisterWithEmail(id, c.Email, token)
		if err != nil {
			out <- err
			return
		}

		out <- repository.Save(executioncontext.ContextWithFlag(ctx, executioncontext.LIVE), u)
	}
}
