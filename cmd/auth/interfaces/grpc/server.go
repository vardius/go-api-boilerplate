/*
Package grpc provides user grpc server
*/
package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/proto"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
	"github.com/vardius/go-api-boilerplate/pkg/jwt"
)

type authenticationServer struct {
	jwt jwt.Jwt
}

// GetToken return token for given email address
func (s *authenticationServer) GetToken(ctx context.Context, req *proto.GetTokenRequest) (*proto.GetTokenResponse, error) {
	// todo get user by email and create token
	id := uuid.New()
	roles := []string{"user"}

	identity := identity.WithValues(id, req.GetEmail(), roles)
	token, e := s.jwt.Encode(identity)

	return &proto.GetTokenResponse{Token: token}, e
}

// RefreshToken return new token based on expired one
func (s *authenticationServer) RefreshToken(ctx context.Context, req *proto.RefreshTokenRequest) (*proto.RefreshTokenResponse, error) {
	identity, e := s.jwt.Decode(req.GetToken())
	if e != nil {
		return new(proto.RefreshTokenResponse), e
	}

	token, e := s.jwt.Encode(identity)

	return &proto.RefreshTokenResponse{Token: token}, e
}

// NewServer returns new user server object
func NewServer(j jwt.Jwt) proto.AuthenticationServer {
	return &authenticationServer{}
}
