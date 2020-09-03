/*
Package grpc provides user grpc server
*/
package grpc

import (
	"context"

	"github.com/google/uuid"
	oauth2models "gopkg.in/oauth2.v4/models"
	"gopkg.in/oauth2.v4/server"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/oauth2"
	"github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	grpcerrors "github.com/vardius/go-api-boilerplate/pkg/grpc/errors"
)

type authenticationServer struct {
	server        *server.Server
	clientStore   *oauth2.ClientStore
	authenticator auth.Authenticator
}

// NewServer returns new auth server object
func NewServer(server *server.Server, clientStore *oauth2.ClientStore, authenticator auth.Authenticator) proto.AuthenticationServiceServer {
	return &authenticationServer{
		server:        server,
		clientStore:   clientStore,
		authenticator: authenticator,
	}
}

// ValidationBearerToken verifies token
func (s *authenticationServer) ValidationBearerToken(ctx context.Context, req *proto.ValidationBearerTokenRequest) (*proto.ValidationBearerTokenResponse, error) {
	if err := s.authenticator.Verify(req.GetToken(), &auth.Claims{}); err != nil {
		return nil, grpcerrors.NewGRPCError(errors.Wrap(err))
	}

	tokenInfo, err := s.server.Manager.LoadAccessToken(ctx, req.GetToken())
	if err != nil {
		return nil, grpcerrors.NewGRPCError(errors.Wrap(err))
	}

	res := &proto.ValidationBearerTokenResponse{
		ClientID: tokenInfo.GetClientID(),
		UserID:   tokenInfo.GetUserID(),
		Scope:    tokenInfo.GetScope(),
	}

	return res, nil
}

// CreateClient verifies token
func (s *authenticationServer) CreateClient(ctx context.Context, req *proto.CreateClientRequest) (*proto.CreateClientResponse, error) {
	clientID := uuid.New().String()
	clientSecret := uuid.New().String()

	// store our internal user service client
	if err := s.clientStore.Set(ctx, &oauth2models.Client{
		ID:     clientID,
		Secret: clientSecret,
		UserID: req.GetUserID(),
		Domain: req.GetDomain(),
	}); err != nil {
		return nil, grpcerrors.NewGRPCError(errors.Wrap(err))
	}

	res := &proto.CreateClientResponse{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		UserID:       req.GetUserID(),
		Domain:       req.GetDomain(),
	}

	return res, nil
}
