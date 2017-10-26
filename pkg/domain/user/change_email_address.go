package user

import (
	"app/pkg/domain"
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

const ChangeEmailAddress = "change-email-address"

type changeEmailAddress struct {
	id    uuid.UUID `json:"id"`
	email string    `json:"email"`
}

func (c *changeEmailAddress) fromJSON(payload json.RawMessage) error {
	return json.Unmarshal(payload, c)
}

func onChangeEmailAddress(repository *eventSourcedRepository) domain.CommandHandler {
	return func(ctx context.Context, payload json.RawMessage, out chan<- error) {
		c := &changeEmailAddress{}
		err := c.fromJSON(payload)
		if err != nil {
			out <- err
			return
		}

		//todo: validate if email is taken

		u := repository.Get(c.id)
		err = u.ChangeEmailAddress(c.email)
		if err != nil {
			out <- err
			return
		}

		out <- nil

		//todo add live flag to context
		repository.Save(ctx, u)
	}
}
