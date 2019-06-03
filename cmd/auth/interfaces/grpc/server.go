/*
Package grpc provides user grpc server
*/
package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/proto"
	"gopkg.in/oauth2.v3/generates"
	"gopkg.in/oauth2.v3/server"
)

type authenticationServer struct {
	server    *server.Server
	secretKey string
}

// VerifyToken verifies token
func (s *authenticationServer) VerifyToken(ctx context.Context, req *proto.VerifyTokenRequest) (res *proto.VerifyTokenResponse, err error) {
	accessToken, err := jwt.ParseWithClaims(req.GetToken(), &generates.JWTAccessClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("parse error")
		}
		return []byte(s.secretKey), nil
	})
	if err != nil {
		return
	}

	_, ok := accessToken.Claims.(*generates.JWTAccessClaims)
	if !ok || !accessToken.Valid {
		return
	}

	tokenInfo, err := s.server.Manager.LoadAccessToken(req.GetToken())

	res = &proto.VerifyTokenResponse{
		ExpiresIn: int64(tokenInfo.GetAccessCreateAt().Add(tokenInfo.GetAccessExpiresIn()).Sub(time.Now()).Seconds()),
		ClientId:  tokenInfo.GetClientID(),
		UserId:    tokenInfo.GetUserID(),
	}

	return
}

// NewServer returns new user server object
func NewServer(server *server.Server, secretKey string) proto.AuthenticationServiceServer {
	return &authenticationServer{
		server,
		secretKey,
	}
}
