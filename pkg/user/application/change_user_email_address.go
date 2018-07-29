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

type changeUserEmailAddress struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

func (c *changeUserEmailAddress) fromJSON(payload json.RawMessage) error {
	return json.Unmarshal(payload, c)
}

// OnChangeUserEmailAddress creates command handler
func OnChangeUserEmailAddress(es eventstore.EventStore, eb eventbus.EventBus) commandbus.CommandHandler {
	repository := infrastructure.NewUserEventSourcedRepository(fmt.Sprintf("%T", user.User{}), es, eb)

	return func(ctx context.Context, payload json.RawMessage, out chan<- error) {
		c := &changeUserEmailAddress{}
		err := c.fromJSON(payload)
		if err != nil {
			out <- err
			return
		}

		//todo: validate if email is taken

		u := repository.Get(c.ID)
		err = u.ChangeEmailAddress(c.Email)
		if err != nil {
			out <- err
			return
		}

		out <- repository.Save(executioncontext.ContextWithFlag(ctx, executioncontext.LIVE), u)
	}
}
