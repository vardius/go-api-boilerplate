package application

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/common/domain"
	"github.com/vardius/go-api-boilerplate/pkg/common/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/common/security/identity"
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
func OnRegisterUserWithEmail(es domain.EventStore, eb domain.EventBus, j jwt.Jwt) domain.CommandHandler {
	repository := infrastructure.New(fmt.Sprintf("%T", user.User{}), es, eb)

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

		out <- nil

		repository.Save(domain.ContextWithFlag(ctx, domain.LIVE), u)
	}
}
