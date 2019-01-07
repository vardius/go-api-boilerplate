/*
Package grpc provides user grpc server
*/
package grpc

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/errors"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventstore"
	"github.com/vardius/go-api-boilerplate/pkg/user/application"
	"github.com/vardius/go-api-boilerplate/pkg/user/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/user/infrastructure/proto"
	"github.com/vardius/go-api-boilerplate/pkg/user/infrastructure/repository"
)

type userServer struct {
	commandBus commandbus.CommandBus
	eventBus   eventbus.EventBus
	eventStore eventstore.EventStore
	jwt        jwt.Jwt
}

// New returns new user server object
func New(cb commandbus.CommandBus, eb eventbus.EventBus, es eventstore.EventStore, j jwt.Jwt) proto.UserServer {
	s := &userServer{cb, eb, es, j}

	registerCommandHandlers(cb, es, eb)
	registerEventHandlers(eb)

	return s
}

// DispatchCommand implements proto.UserServer interface
func (s *userServer) DispatchCommand(ctx context.Context, cmd *proto.DispatchCommandRequest) (*empty.Empty, error) {
	out := make(chan error)
	defer close(out)

	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				out <- errors.Newf(errors.INTERNAL, "Recovered in f %v", rec)
			}
		}()

		c, err := buildDomainCommand(ctx, cmd)
		if err != nil {
			out <- err
			return
		}

		s.commandBus.Publish(ctx, fmt.Sprintf("%T", c), c, out)
	}()

	select {
	case <-ctx.Done():
		return new(empty.Empty), ctx.Err()
	case err := <-out:
		return new(empty.Empty), err
	}
}

func registerCommandHandlers(cb commandbus.CommandBus, es eventstore.EventStore, eb eventbus.EventBus) {
	repository := repository.NewUserRepository(es, eb)

	cb.Subscribe(fmt.Sprintf("%T", &user.RegisterWithEmail{}), user.OnRegisterWithEmail(repository))
	cb.Subscribe(fmt.Sprintf("%T", &user.RegisterWithGoogle{}), user.OnRegisterWithGoogle(repository))
	cb.Subscribe(fmt.Sprintf("%T", &user.RegisterWithFacebook{}), user.OnRegisterWithFacebook(repository))
	cb.Subscribe(fmt.Sprintf("%T", &user.ChangeEmailAddress{}), user.OnChangeEmailAddress(repository))
}

func registerEventHandlers(eb eventbus.EventBus) {
	eb.Subscribe(fmt.Sprintf("%T", &user.WasRegisteredWithEmail{}), application.WhenUserWasRegisteredWithEmail)
	eb.Subscribe(fmt.Sprintf("%T", &user.WasRegisteredWithGoogle{}), application.WhenUserWasRegisteredWithGoogle)
	eb.Subscribe(fmt.Sprintf("%T", &user.WasRegisteredWithFacebook{}), application.WhenUserWasRegisteredWithFacebook)
	eb.Subscribe(fmt.Sprintf("%T", &user.EmailAddressWasChanged{}), application.WhenUserEmailAddressWasChanged)
}
