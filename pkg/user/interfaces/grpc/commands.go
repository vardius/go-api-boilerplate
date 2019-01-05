/*
Package grpc provides user grpc server
*/
package grpc

import (
	"context"
	"encoding/json"

	"github.com/vardius/go-api-boilerplate/pkg/common/application/errors"
	"github.com/vardius/go-api-boilerplate/pkg/user/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/user/infrastructure/proto"
)

// ChangeUserEmailAddress command bus contract
const ChangeUserEmailAddress = "change-user-email-address"

// RegisterUserWithEmail command bus contract
const RegisterUserWithEmail = "register-user-with-email"

// RegisterUserWithFacebook command bus contract
const RegisterUserWithFacebook = "register-user-with-facebook"

// RegisterUserWithGoogle command bus contract
const RegisterUserWithGoogle = "register-user-with-google"

func buildDomainCommand(ctx context.Context, cmd *proto.DispatchCommandRequest) (interface{}, error) {
	var c interface{}
	switch cmd.GetName() {
	case RegisterUserWithEmail:
		c = &user.RegisterWithEmail{}
	case RegisterUserWithGoogle:
		c = &user.RegisterWithGoogle{}
	case RegisterUserWithFacebook:
		c = &user.RegisterWithFacebook{}
	case ChangeUserEmailAddress:
		c = &user.ChangeEmailAddress{}
	default:
		return nil, errors.New(errors.INTERNAL, "Invalid command")
	}

	err := json.Unmarshal(cmd.GetPayload(), c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
