/*
Package grpc provides user grpc server
*/
package grpc

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/cmd/user/proto"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	grpcerrors "github.com/vardius/go-api-boilerplate/pkg/grpc/errors"
)

type userServer struct {
	userRepository persistence.UserRepository
}

// NewServer returns new user server object
func NewServer(r persistence.UserRepository) proto.UserServiceServer {
	s := &userServer{
		userRepository: r,
	}

	return s
}

// DispatchUserCommand implements proto.UserServiceServer interface.
// It always fails: the gRPC server trusts identity from unsigned metadata, so a
// command sent here could act as any user. Clients use the HTTP API instead.
func (s *userServer) DispatchUserCommand(context.Context, *proto.DispatchUserCommandRequest) (*empty.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "user commands are accepted only over HTTP")
}

// GetUser implements proto.UserServiceServer interface
func (s *userServer) GetUser(ctx context.Context, r *proto.GetUserRequest) (*proto.User, error) {
	u, err := s.userRepository.Get(ctx, r.GetId())
	if err != nil {
		return nil, grpcerrors.NewGRPCError(apperrors.Wrap(err))
	}

	return &proto.User{
		Id:         u.GetID(),
		Email:      u.GetEmail(),
		FacebookId: u.GetFacebookID(),
		GoogleId:   u.GetGoogleID(),
	}, nil
}

// ListUsers implements proto.UserServiceServer interface
func (s *userServer) ListUsers(ctx context.Context, r *proto.ListUserRequest) (*proto.ListUserResponse, error) {
	if r.GetPage() < 1 || r.GetLimit() < 1 {
		return nil, grpcerrors.NewGRPCError(apperrors.New("Invalid page or limit value. Please provide values greater then 1"))
	}

	var users []persistence.User
	var list []*proto.User

	totalUsers, err := s.userRepository.Count(ctx)
	if err != nil {
		return nil, grpcerrors.NewGRPCError(apperrors.Wrap(err))
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
		return nil, grpcerrors.NewGRPCError(apperrors.Wrap(err))
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
