/*
Package grpc provides user grpc server
*/
package grpc

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/vardius/go-api-boilerplate/cmd/user/application"
	"github.com/vardius/go-api-boilerplate/cmd/user/domain/user"
	"github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/persistence/mysql"
	"github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/proto"
	"github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/repository"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/eventstore"
	"github.com/vardius/go-api-boilerplate/pkg/jwt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type userServer struct {
	commandBus commandbus.CommandBus
	eventBus   eventbus.EventBus
	eventStore eventstore.EventStore
	db         *sql.DB
	jwt        jwt.Jwt
}

// NewServer returns new user server object
func NewServer(cb commandbus.CommandBus, eb eventbus.EventBus, es eventstore.EventStore, db *sql.DB, j jwt.Jwt) proto.UserServiceServer {
	s := &userServer{cb, eb, es, db, j}

	userRepository := repository.NewUserRepository(es, eb)
	userMYSQLRepository := mysql.NewUserRepository(db)

	s.registerCommandHandlers(userRepository)
	s.registerEventHandlers(userMYSQLRepository)

	return s
}

// DispatchCommand implements proto.UserServiceServer interface
func (s *userServer) DispatchCommand(ctx context.Context, r *proto.DispatchCommandRequest) (*empty.Empty, error) {
	c, err := buildDomainCommand(ctx, r.GetName(), r.GetPayload())
	if err != nil {
		return new(empty.Empty), err
	}

	out := make(chan error)
	defer close(out)

	go func() {
		s.commandBus.Publish(ctx, fmt.Sprintf("%T", c), c, out)
	}()

	select {
	case <-ctx.Done():
		return new(empty.Empty), ctx.Err()
	case err := <-out:
		return new(empty.Empty), err
	}
}

// GetUser implements proto.UserServiceServer interface
func (s *userServer) GetUser(ctx context.Context, r *proto.GetUserRequest) (*proto.User, error) {
	repository := mysql.NewUserRepository(s.db)

	user, err := repository.Get(ctx, r.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "User not found")
	}

	return user, nil
}

// ListUsers implements proto.UserServiceServer interface
func (s *userServer) ListUsers(ctx context.Context, r *proto.ListUserRequest) (*proto.ListUserResponse, error) {
	if r.GetPage() < 1 || r.GetLimit() < 1 {
		return nil, status.Error(codes.Internal, "Invalid page or limit value. Please provide values greater then 1")
	}

	var users []*proto.User

	repository := mysql.NewUserRepository(s.db)

	totalUsers, err := repository.Count(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to count users")
	}

	offset := (r.GetPage() * r.GetLimit()) - r.GetLimit()

	if totalUsers < 1 || offset > (totalUsers-1) {
		return &proto.ListUserResponse{
			Page:  r.GetPage(),
			Limit: r.GetLimit(),
			Total: totalUsers,
			Users: users,
		}, nil
	}

	users, err = repository.FindAll(ctx, r.GetLimit(), offset)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to fetch users")
	}

	response := &proto.ListUserResponse{
		Page:  r.GetPage(),
		Limit: r.GetLimit(),
		Total: totalUsers,
		Users: users,
	}

	return response, nil
}

func (s *userServer) registerCommandHandlers(r user.Repository) {
	s.commandBus.Subscribe(fmt.Sprintf("%T", &user.RegisterWithEmail{}), user.OnRegisterWithEmail(r, s.db))
	s.commandBus.Subscribe(fmt.Sprintf("%T", &user.RegisterWithGoogle{}), user.OnRegisterWithGoogle(r, s.db))
	s.commandBus.Subscribe(fmt.Sprintf("%T", &user.RegisterWithFacebook{}), user.OnRegisterWithFacebook(r, s.db))
	s.commandBus.Subscribe(fmt.Sprintf("%T", &user.ChangeEmailAddress{}), user.OnChangeEmailAddress(r, s.db))
}

func (s *userServer) registerEventHandlers(r persistence.UserRepository) {
	s.eventBus.Subscribe(fmt.Sprintf("%T", &user.WasRegisteredWithEmail{}), application.WhenUserWasRegisteredWithEmail(s.db, r))
	s.eventBus.Subscribe(fmt.Sprintf("%T", &user.WasRegisteredWithGoogle{}), application.WhenUserWasRegisteredWithGoogle(s.db, r))
	s.eventBus.Subscribe(fmt.Sprintf("%T", &user.WasRegisteredWithFacebook{}), application.WhenUserWasRegisteredWithFacebook(s.db, r))
	s.eventBus.Subscribe(fmt.Sprintf("%T", &user.EmailAddressWasChanged{}), application.WhenUserEmailAddressWasChanged(s.db, r))
}
