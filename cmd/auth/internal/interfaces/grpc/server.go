/*
Package grpc provides user grpc server
*/
package grpc

import (
	"context"
	"time"

	"github.com/dgrijalva/jwt-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/oauth2.v3/generates"
	"gopkg.in/oauth2.v3/server"

	"github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

type authenticationServer struct {
	server    *server.Server
	logger    *log.Logger
	secretKey string
}

// NewServer returns new user server object
func NewServer(server *server.Server, logger *log.Logger, secretKey string) proto.AuthenticationServiceServer {
	return &authenticationServer{
		server,
		logger,
		secretKey,
	}
}

// VerifyToken verifies token
func (s *authenticationServer) VerifyToken(ctx context.Context, req *proto.VerifyTokenRequest) (*proto.VerifyTokenResponse, error) {
	accessToken, err := jwt.ParseWithClaims(req.GetToken(), &generates.JWTAccessClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.Newf(errors.INTERNAL, "Failed to decode token, invalid signing method")
		}
		return []byte(s.secretKey), nil
	})
	if err != nil {
		s.logger.Error(ctx, "%v\n", errors.Wrap(err, errors.INTERNAL, "Token parse error"))
		return nil, status.Error(codes.Internal, "Failed to parse token with claims")
	}

	_, ok := accessToken.Claims.(*generates.JWTAccessClaims)
	if !ok || !accessToken.Valid {
		s.logger.Error(ctx, "%v\n", errors.New(errors.INTERNAL, "Token is not valid, could not parse claims"))
		return nil, status.Error(codes.Internal, "Token is not valid, could not parse claims")
	}

	tokenInfo, err := s.server.Manager.LoadAccessToken(req.GetToken())
	if err != nil {
		s.logger.Error(ctx, "%v\n", errors.Wrap(err, errors.NOTFOUND, "Could not load token"))
		return nil, status.Error(codes.NotFound, "Could not load token")
	}

	res := &proto.VerifyTokenResponse{
		ExpiresIn: int64(tokenInfo.GetAccessCreateAt().Add(tokenInfo.GetAccessExpiresIn()).Sub(time.Now()).Seconds()),
		ClientId:  tokenInfo.GetClientID(),
		UserId:    tokenInfo.GetUserID(),
	}

	return res, nil
}
