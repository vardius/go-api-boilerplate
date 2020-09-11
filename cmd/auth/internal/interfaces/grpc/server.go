/*
Package grpc provides user grpc server
*/
package grpc

import (
	"context"

	"github.com/google/uuid"
	"gopkg.in/oauth2.v4/server"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/client"
	"github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
	grpcerrors "github.com/vardius/go-api-boilerplate/pkg/grpc/errors"
)

type authenticationServer struct {
	server                 *server.Server
	eventSourcedRepository client.Repository
	authenticator          auth.Authenticator
}

// NewServer returns new auth server object
func NewServer(server *server.Server, eventSourcedRepository client.Repository, authenticator auth.Authenticator) proto.AuthenticationServiceServer {
	return &authenticationServer{
		server:                 server,
		eventSourcedRepository: eventSourcedRepository,
		authenticator:          authenticator,
	}
}

// ValidationBearerToken verifies token
func (s *authenticationServer) ValidationBearerToken(ctx context.Context, req *proto.ValidationBearerTokenRequest) (*proto.ValidationBearerTokenResponse, error) {
	if err := s.authenticator.Verify(req.GetToken(), &auth.Claims{}); err != nil {
		return nil, grpcerrors.NewGRPCError(apperrors.Wrap(err))
	}

	tokenInfo, err := s.server.Manager.LoadAccessToken(ctx, req.GetToken())
	if err != nil {
		return nil, grpcerrors.NewGRPCError(apperrors.Wrap(err))
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
	clientID := uuid.New()
	clientSecret := uuid.New()

	userID, err := uuid.Parse(req.GetUserID())
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	c := client.New()
	if err := c.Create(ctx, clientID, clientSecret, userID, req.GetDomain(), req.GetRedirectURL(), req.GetScopes()...); err != nil {
		return nil, apperrors.Wrap(err)
	}

	// we block here until event handler is done
	// this is because when other services request access token after creating client
	// we want handler to be finished and client persisted in storage
	if err := s.eventSourcedRepository.SaveAndAcknowledge(executioncontext.WithFlag(ctx, executioncontext.LIVE), c); err != nil {
		return nil, apperrors.Wrap(err)
	}

	res := &proto.CreateClientResponse{
		ClientID:     clientID.String(),
		ClientSecret: clientSecret.String(),
		UserID:       req.GetUserID(),
		Domain:       req.GetDomain(),
		RedirectURL:  req.GetRedirectURL(),
		Scopes:       req.GetScopes(),
	}

	return res, nil
}
