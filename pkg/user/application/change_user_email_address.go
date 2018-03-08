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

// ChangeUserEmailAddress command bus contract
const ChangeUserEmailAddress = "change-user-email-address"

type changeUserEmailAddress struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

func (c *changeUserEmailAddress) fromJSON(payload json.RawMessage) error {
	return json.Unmarshal(payload, c)
}

// OnChangeUserEmailAddress creates command handler
func OnChangeUserEmailAddress(es domain.EventStore, eb domain.EventBus) domain.CommandHandler {
	repository := infrastructure.New(fmt.Sprintf("%T", user.User{}), es, eb)

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

		out <- nil

		repository.Save(domain.ContextWithFlag(ctx, domain.LIVE), u)
	}
}
