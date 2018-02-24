package userserver

import (
	"context"
	"fmt"

	"github.com/vardius/go-api-boilerplate/internal/user"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/jwt"
	pb "github.com/vardius/go-api-boilerplate/rpc/domain"
)

func registerCommandHandlers(cb domain.CommandBus, es domain.EventStore, eb domain.EventBus, j jwt.Jwt) {
	cb.Subscribe(user.RegisterWithEmail, user.OnRegisterWithEmail(es, eb, j))
	cb.Subscribe(user.RegisterWithGoogle, user.OnRegisterWithGoogle(es, eb))
	cb.Subscribe(user.RegisterWithFacebook, user.OnRegisterWithFacebook(es, eb))
	cb.Subscribe(user.ChangeEmailAddress, user.OnChangeEmailAddress(es, eb))
}

func registerEventHandlers(eb domain.EventBus) {
	eb.Subscribe(fmt.Sprintf("%T", &user.WasRegisteredWithEmail{}), user.WhenWasRegisteredWithEmail)
	eb.Subscribe(fmt.Sprintf("%T", &user.WasRegisteredWithGoogle{}), user.WhenWasRegisteredWithGoogle)
	eb.Subscribe(fmt.Sprintf("%T", &user.WasRegisteredWithFacebook{}), user.WhenWasRegisteredWithFacebook)
	eb.Subscribe(fmt.Sprintf("%T", &user.EmailAddressWasChanged{}), user.WhenEmailAddressWasChanged)
}

type userServer struct {
	commandBus domain.CommandBus
	eventBus   domain.EventBus
	eventStore domain.EventStore
	jwt        jwt.Jwt
}

// Dispatch implements pb.DomainServer interface
func (s *userServer) Dispatch(ctx context.Context, cmd *pb.Command) (*pb.Response, error) {
	out := make(chan error)
	defer close(out)

	go func() {
		s.commandBus.Publish(ctx, cmd.GetName(), cmd.GetPayload(), out)
	}()

	if err := <-out; err != nil {
		return new(pb.Response), err
	}

	return new(pb.Response), nil
}

// New returns new user domain server object
func New(cb domain.CommandBus, eb domain.EventBus, es domain.EventStore, j jwt.Jwt) pb.DomainServer {
	s := &userServer{cb, eb, es, j}

	registerCommandHandlers(cb, es, eb, j)
	registerEventHandlers(eb)

	return s
}
