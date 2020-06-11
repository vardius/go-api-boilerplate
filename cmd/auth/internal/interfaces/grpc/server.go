/*
Package grpc provides user grpc server
*/
package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/oauth2.v4/server"

	"github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

type authenticationServer struct {
	server        *server.Server
	authenticator auth.Authenticator
	logger        *log.Logger
}

// NewServer returns new user server object
func NewServer(server *server.Server, authenticator auth.Authenticator, logger *log.Logger) proto.AuthenticationServiceServer {
	return &authenticationServer{
		server,
		authenticator,
		logger,
	}
}

// VerifyToken verifies token
func (s *authenticationServer) VerifyToken(ctx context.Context, req *proto.VerifyTokenRequest) (*proto.VerifyTokenResponse, error) {
	if err := s.authenticator.Verify(req.GetToken(), &auth.Claims{}); err != nil {
		s.logger.Error(ctx, "%v\n", err)
		return nil, status.Error(codes.Internal, "Invalid token, could not verify")
	}

	tokenInfo, err := s.server.Manager.LoadAccessToken(ctx, req.GetToken())
	if err != nil {
		s.logger.Error(ctx, "%v\n", err)
		return nil, status.Error(codes.NotFound, "Could not load token info")
	}

	res := &proto.VerifyTokenResponse{
		ExpiresIn: int64(tokenInfo.GetAccessCreateAt().Add(tokenInfo.GetAccessExpiresIn()).Sub(time.Now()).Seconds()),
		ClientId:  tokenInfo.GetClientID(),
		UserId:    tokenInfo.GetUserID(),
	}

	return res, nil
}
