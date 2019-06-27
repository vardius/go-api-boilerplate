/*
Package grpc provides user grpc server
*/
package grpc

import (
	"context"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/proto"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"gopkg.in/oauth2.v3/generates"
	"gopkg.in/oauth2.v3/server"
)

type authenticationServer struct {
	server    *server.Server
	secretKey string
}

// VerifyToken verifies token
func (s *authenticationServer) VerifyToken(ctx context.Context, req *proto.VerifyTokenRequest) (*proto.VerifyTokenResponse, error) {
	accessToken, err := jwt.ParseWithClaims(req.GetToken(), &generates.JWTAccessClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.Newf(errors.INTERNAL, "parse error")
		}
		return []byte(s.secretKey), nil
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.INTERNAL, "Token parse error")
	}

	_, ok := accessToken.Claims.(*generates.JWTAccessClaims)
	if !ok || !accessToken.Valid {
		return nil, errors.New(errors.INTERNAL, "Token is not valid, could not parse claims")
	}

	tokenInfo, err := s.server.Manager.LoadAccessToken(req.GetToken())

	res := &proto.VerifyTokenResponse{
		ExpiresIn: int64(tokenInfo.GetAccessCreateAt().Add(tokenInfo.GetAccessExpiresIn()).Sub(time.Now()).Seconds()),
		ClientId:  tokenInfo.GetClientID(),
		UserId:    tokenInfo.GetUserID(),
	}

	return res, errors.Wrap(err, errors.NOTFOUND, "Could not load token")
}

// NewServer returns new user server object
func NewServer(server *server.Server, secretKey string) proto.AuthenticationServiceServer {
	return &authenticationServer{
		server,
		secretKey,
	}
}
