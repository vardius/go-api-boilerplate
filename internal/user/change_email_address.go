package user

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

// ChangeEmailAddress command bus contract
const ChangeEmailAddress = "change-user-email-address"

type changeEmailAddress struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

func (c *changeEmailAddress) fromJSON(payload json.RawMessage) error {
	return json.Unmarshal(payload, c)
}

// OnChangeEmailAddress creates command handler
func OnChangeEmailAddress(es domain.EventStore, eb domain.EventBus) domain.CommandHandler {
	repository := newRepository(fmt.Sprintf("%T", User{}), es, eb)

	return func(ctx context.Context, payload json.RawMessage, out chan<- error) {
		c := &changeEmailAddress{}
		err := c.fromJSON(payload)
		if err != nil {
			out <- err
			return
		}

		//todo: validate if email is taken

		user := repository.Get(c.ID)
		err = user.ChangeEmailAddress(c.Email)
		if err != nil {
			out <- err
			return
		}

		out <- nil

		repository.Save(domain.ContextWithFlag(ctx, domain.LIVE), user)
	}
}
