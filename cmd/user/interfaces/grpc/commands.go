/*
Package grpc provides user grpc server
*/
package grpc

import (
	"context"
	"encoding/json"

	"github.com/vardius/go-api-boilerplate/cmd/user/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

// RequestUserAccessToken command bus contract
const RequestUserAccessToken = "request-user-access-token"

// ChangeUserEmailAddress command bus contract
const ChangeUserEmailAddress = "change-user-email-address"

// RegisterUserWithEmail command bus contract
const RegisterUserWithEmail = "register-user-with-email"

// RegisterUserWithFacebook command bus contract
const RegisterUserWithFacebook = "register-user-with-facebook"

// RegisterUserWithGoogle command bus contract
const RegisterUserWithGoogle = "register-user-with-google"

func buildDomainCommand(ctx context.Context, name string, payload []byte) (interface{}, error) {
	var c interface{}
	switch name {
	case RegisterUserWithEmail:
		c = &user.RegisterWithEmail{}
	case RegisterUserWithGoogle:
		c = &user.RegisterWithGoogle{}
	case RegisterUserWithFacebook:
		c = &user.RegisterWithFacebook{}
	case ChangeUserEmailAddress:
		c = &user.ChangeEmailAddress{}
	case RequestUserAccessToken:
		c = &user.RequestAccessToken{}
	default:
		return nil, errors.New(errors.INTERNAL, "Invalid command")
	}

	err := json.Unmarshal(payload, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
