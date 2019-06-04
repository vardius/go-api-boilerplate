/*
Package grpc provides user grpc server
*/
package grpc

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/persistence/mysql"
	"github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/proto"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type userServer struct {
	commandBus commandbus.CommandBus
	db         *sql.DB
}

// NewServer returns new user server object
func NewServer(cb commandbus.CommandBus, db *sql.DB) proto.UserServiceServer {
	s := &userServer{cb, db}

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
