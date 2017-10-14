package user

import (
	"app/pkg/auth"
	"app/pkg/domain"
	"app/pkg/domain/user/command"
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

const Domain = "users"

const RegisterUserWithEmail string = "register-user-with-email"
const RegisterUserWithGoogle string = "register-user-with-google"
const RegisterUserWithFacebook string = "register-user-with-facebook"
const ChangeUserEmailAddress string = "change-user-email-address"

func registerCommandHandlers(commandBus domain.CommandBus, repository *eventSourcedRepository, jwtService auth.JwtService) {
	commandBus.Subscribe(Domain+"-"+RegisterUserWithEmail, registerUserWithEmail(repository, jwtService))
	commandBus.Subscribe(Domain+"-"+RegisterUserWithGoogle, registerUserWithGoogle(repository))
	commandBus.Subscribe(Domain+"-"+RegisterUserWithFacebook, registerUserWithFacebook(repository))
	commandBus.Subscribe(Domain+"-"+ChangeUserEmailAddress, changeUserEmailAddress(repository))
}

func registerUserWithEmail(repository *eventSourcedRepository, jwtService auth.JwtService) domain.CommandHandler {
	return func(ctx context.Context, payload json.RawMessage, out chan<- error) {
		c, err := command.NewRegisterUserWithEmail(payload)
		if err != nil {
			out <- err
			return
		}

		//todo: validate if email is taken

		id, err := uuid.NewRandom()
		if err != nil {
			out <- err
			return
		}

		identity := auth.NewUserIdentity(id, c.Email)
		token, err := jwtService.GenerateToken(identity)
		if err != nil {
			out <- err
			return
		}

		user := New()
		err = user.registerUserWithEmail(id, c.Email, token)
		if err != nil {
			out <- err
			return
		}

		out <- nil

		// todo add live flag to context
		repository.Save(ctx, user)
	}
}

func registerUserWithGoogle(repository *eventSourcedRepository) domain.CommandHandler {
	return func(ctx context.Context, payload json.RawMessage, out chan<- error) {
		c, err := command.NewRegisterUserWithGoogle(payload)
		if err != nil {
			out <- err
			return
		}

		//todo: validate if email is taken or if user already connected with google

		id, err := uuid.NewRandom()
		if err != nil {
			out <- err
			return
		}

		user := New()
		err = user.registerUserWithGoogle(id, c.Email, c.AuthToken)
		if err != nil {
			out <- err
			return
		}

		out <- nil

		// todo add live flag to context
		repository.Save(ctx, user)
	}
}

func registerUserWithFacebook(repository *eventSourcedRepository) domain.CommandHandler {
	return func(ctx context.Context, payload json.RawMessage, out chan<- error) {
		c, err := command.NewRegisterUserWithFacebook(payload)
		if err != nil {
			out <- err
			return
		}

		//todo: validate if email is taken or if user already connected with facebook

		id, err := uuid.NewRandom()
		if err != nil {
			out <- err
			return
		}

		user := New()
		err = user.registerUserWithFacebook(id, c.Email, c.AuthToken)
		if err != nil {
			out <- err
			return
		}

		out <- nil

		// todo add live flag to context
		repository.Save(ctx, user)
	}
}

func changeUserEmailAddress(repository *eventSourcedRepository) domain.CommandHandler {
	return func(ctx context.Context, payload json.RawMessage, out chan<- error) {
		c, err := command.NewChangeUserEmailAddress(payload)
		if err != nil {
			out <- err
			return
		}

		//todo: validate if email is taken

		user := repository.Get(c.Id)
		err = user.changeEmailAddress(c.Email)
		if err != nil {
			out <- err
			return
		}

		out <- nil

		//todo add live flag to context
		repository.Save(ctx, user)
	}
}
