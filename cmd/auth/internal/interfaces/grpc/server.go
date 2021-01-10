/*
Package grpc provides auth grpc server
*/
package grpc

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"gopkg.in/oauth2.v4/server"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/client"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	"github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	grpcerrors "github.com/vardius/go-api-boilerplate/pkg/grpc/errors"
)

type authenticationServer struct {
	server     *server.Server
	commandBus commandbus.CommandBus
}

// NewServer returns new auth server object
func NewServer(server *server.Server, commandBus commandbus.CommandBus) proto.AuthenticationServiceServer {
	return &authenticationServer{
		server:     server,
		commandBus: commandBus,
	}
}

// DispatchTokenCommand dispatches token commands
func (s *authenticationServer) DispatchTokenCommand(ctx context.Context, r *proto.DispatchAuthCommandRequest) (*empty.Empty, error) {
	c, err := token.NewCommandFromPayload(r.GetName(), r.GetPayload())
	if err != nil {
		return nil, grpcerrors.NewGRPCError(apperrors.Wrap(err))
	}

	if err := s.commandBus.Publish(ctx, c); err != nil {
		return nil, grpcerrors.NewGRPCError(apperrors.Wrap(err))
	}

	return new(empty.Empty), nil
}

// DispatchClientCommand dispatches client commands
func (s *authenticationServer) DispatchClientCommand(ctx context.Context, r *proto.DispatchAuthCommandRequest) (*empty.Empty, error) {
	c, err := client.NewCommandFromPayload(r.GetName(), r.GetPayload())
	if err != nil {
		return nil, grpcerrors.NewGRPCError(apperrors.Wrap(err))
	}

	if err := s.commandBus.Publish(ctx, c); err != nil {
		return nil, grpcerrors.NewGRPCError(apperrors.Wrap(err))
	}

	return new(empty.Empty), nil
}

// ValidationBearerToken verifies token existence
func (s *authenticationServer) ValidationBearerToken(ctx context.Context, req *proto.ValidationBearerTokenRequest) (*empty.Empty, error) {
	if _, err := s.server.Manager.LoadAccessToken(ctx, req.GetToken()); err != nil {
		return nil, grpcerrors.NewGRPCError(apperrors.Wrap(fmt.Errorf("failed to load token (%s): %w", req.GetToken(), err)))
	}

	return new(empty.Empty), nil
}
