/*
Package grpc provides user grpc server
*/
package grpc

import (
	"context"

	"github.com/vardius/go-api-boilerplate/pkg/common/application/security/identity"
	"github.com/vardius/go-api-boilerplate/pkg/user/infrastructure/proto"
)

type authenticator struct {
	username string
	password string
	token    string
}

// GetToken return token for given email address
func (a *authenticator) GetToken(ctx context.Context, req *proto.GetTokenRequest) (*proto.GetTokenResponse, error) {
	// todo get user by email and create token

	identity := &identity.Identity{}
	identity.FromGoogleData(data)

	token, e := g.jwt.Encode(identity)

	return &proto.GetTokenResponse{token}, e
}

// RefreshToken return new token based on expired one
func (a *authenticator) RefreshToken(ctx context.Context, req *proto.RefreshTokenRequest) (*proto.RefreshTokenResponse, error) {
	// todo decode token and generat new one

	identity := &identity.Identity{}
	identity.FromGoogleData(data)

	token, e := g.jwt.Encode(identity)

	return &proto.RefreshTokenResponse{token}, e
}

// New returns new user server object
func New() proto.UserServer {
	a := &authenticator{}

	return a
}
