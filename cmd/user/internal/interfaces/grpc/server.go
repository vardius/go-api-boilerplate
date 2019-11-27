/*
Package grpc provides user grpc server
*/
package grpc

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/cmd/user/proto"
	"github.com/vardius/go-api-boilerplate/internal/commandbus"
	"github.com/vardius/go-api-boilerplate/internal/errors"
	"github.com/vardius/go-api-boilerplate/internal/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type userServer struct {
	commandBus     commandbus.CommandBus
	userRepository persistence.UserRepository
	logger         *log.Logger
}

// NewServer returns new user server object
func NewServer(cb commandbus.CommandBus, r persistence.UserRepository, l *log.Logger) proto.UserServiceServer {
	s := &userServer{cb, r, l}

	return s
}

// DispatchCommand implements proto.UserServiceServer interface
func (s *userServer) DispatchCommand(ctx context.Context, r *proto.DispatchCommandRequest) (*empty.Empty, error) {
	c, err := user.NewCommandFromPayload(r.GetName(), r.GetPayload())
	if err != nil {
		s.logger.Error(ctx, "%v\n", errors.Wrap(err, errors.INTERNAL, "Could not build command from payload"))
		return nil, status.Error(codes.Internal, "Could not build command from payload")
	}

	out := make(chan error, 1)
	defer close(out)

	go func() {
		s.commandBus.Publish(ctx, c, out)
	}()

	ctxDoneCh := ctx.Done()
	select {
	case <-ctxDoneCh:
		return nil, status.Error(codes.Internal, "Context done")
	case err := <-out:
		if err != nil {
			s.logger.Error(ctx, "%v\n", errors.Wrap(err, errors.INTERNAL, "Publish command error"))
			return nil, status.Error(codes.Internal, "Publish command error")
		}

		return new(empty.Empty), nil
	}
}

// GetUser implements proto.UserServiceServer interface
func (s *userServer) GetUser(ctx context.Context, r *proto.GetUserRequest) (*proto.User, error) {
	user, err := s.userRepository.Get(ctx, r.GetId())
	if err != nil {
		s.logger.Error(ctx, "%v\n", errors.Wrap(err, errors.NOTFOUND, "User not found"))
		return nil, status.Error(codes.NotFound, "User not found")
	}

	return &proto.User{
		Id:         user.GetID(),
		Email:      user.GetEmail(),
		FacebookId: user.GetFacebookID(),
		GoogleId:   user.GetGoogleID(),
	}, nil
}

// ListUsers implements proto.UserServiceServer interface
func (s *userServer) ListUsers(ctx context.Context, r *proto.ListUserRequest) (*proto.ListUserResponse, error) {
	if r.GetPage() < 1 || r.GetLimit() < 1 {
		return nil, status.Error(codes.Internal, "Invalid page or limit value. Please provide values greater then 1")
	}

	var users []persistence.User
	var list []*proto.User

	totalUsers, err := s.userRepository.Count(ctx)
	if err != nil {
		s.logger.Error(ctx, "%v\n", errors.Wrap(err, errors.INTERNAL, "Failed to count users"))
		return nil, status.Error(codes.Internal, "Failed to count users")
	}

	offset := (r.GetPage() * r.GetLimit()) - r.GetLimit()

	if totalUsers < 1 || offset > (totalUsers-1) {
		return &proto.ListUserResponse{
			Page:  r.GetPage(),
			Limit: r.GetLimit(),
			Total: totalUsers,
			Users: list,
		}, nil
	}

	users, err = s.userRepository.FindAll(ctx, r.GetLimit(), offset)
	if err != nil {
		s.logger.Error(ctx, "%v\n", errors.Wrap(err, errors.INTERNAL, "Failed to fetch users"))
		return nil, status.Error(codes.Internal, "Failed to fetch users")
	}

	list = make([]*proto.User, len(users))
	for i := range users {
		list[i] = &proto.User{
			Id:         users[i].GetID(),
			Email:      users[i].GetEmail(),
			FacebookId: users[i].GetFacebookID(),
			GoogleId:   users[i].GetGoogleID(),
		}
	}

	response := &proto.ListUserResponse{
		Page:  r.GetPage(),
		Limit: r.GetLimit(),
		Total: totalUsers,
		Users: list,
	}

	return response, nil
}
