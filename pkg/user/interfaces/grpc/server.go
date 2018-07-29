/*
Package grpc provides user grpc server
*/
package grpc

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventstore"
	"github.com/vardius/go-api-boilerplate/pkg/user/application"
	"github.com/vardius/go-api-boilerplate/pkg/user/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/user/infrastructure/proto"
)

func registerCommandHandlers(cb commandbus.CommandBus, es eventstore.EventStore, eb eventbus.EventBus, j jwt.Jwt) {
	cb.Subscribe(RegisterUserWithEmail, application.OnRegisterUserWithEmail(es, eb, j))
	cb.Subscribe(RegisterUserWithGoogle, application.OnRegisterUserWithGoogle(es, eb))
	cb.Subscribe(RegisterUserWithFacebook, application.OnRegisterUserWithFacebook(es, eb))
	cb.Subscribe(ChangeUserEmailAddress, application.OnChangeUserEmailAddress(es, eb))
}

func registerEventHandlers(eb eventbus.EventBus) {
	eb.Subscribe(fmt.Sprintf("%T", &user.WasRegisteredWithEmail{}), application.WhenUserWasRegisteredWithEmail)
	eb.Subscribe(fmt.Sprintf("%T", &user.WasRegisteredWithGoogle{}), application.WhenUserWasRegisteredWithGoogle)
	eb.Subscribe(fmt.Sprintf("%T", &user.WasRegisteredWithFacebook{}), application.WhenUserWasRegisteredWithFacebook)
	eb.Subscribe(fmt.Sprintf("%T", &user.EmailAddressWasChanged{}), application.WhenUserEmailAddressWasChanged)
}

type userServer struct {
	commandBus commandbus.CommandBus
	eventBus   eventbus.EventBus
	eventStore eventstore.EventStore
	jwt        jwt.Jwt
}

// DispatchCommand implements proto.UserServer interface
func (s *userServer) DispatchCommand(ctx context.Context, cmd *proto.DispatchCommandRequest) error {
	out := make(chan error)
	defer close(out)

	go func() {
		s.commandBus.Publish(ctx, cmd.GetName(), cmd.GetPayload(), out)
	}()

	select {
	case <-ctx.Done():
		return new(empty.Empty), ctx.Err()
	case err := <-out:
		return new(empty.Empty), err
	}
}

// New returns new user server object
func New(cb commandbus.CommandBus, eb eventbus.EventBus, es eventstore.EventStore, j jwt.Jwt) proto.UserServer {
	s := &userServer{cb, eb, es, j}

	registerCommandHandlers(cb, es, eb, j)
	registerEventHandlers(eb)

	return s
}
