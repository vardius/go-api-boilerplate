package domain

import (
	"context"
	"github.com/dgrijalva/jwt-go"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/eventhandler"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/services"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
)

func RegisterUserDomain(ctx context.Context, cfg *config.Config, container *services.ServiceContainer) error {
	if err := domain.RegisterEventFactory(user.WasRegisteredWithEmailType, func() interface{} { return &user.WasRegisteredWithEmail{} }); err != nil {
		return apperrors.Wrap(err)
	}
	if err := domain.RegisterEventFactory(user.WasRegisteredWithGoogleType, func() interface{} { return &user.WasRegisteredWithGoogle{} }); err != nil {
		return apperrors.Wrap(err)
	}
	if err := domain.RegisterEventFactory(user.WasRegisteredWithFacebookType, func() interface{} { return &user.WasRegisteredWithFacebook{} }); err != nil {
		return apperrors.Wrap(err)
	}
	if err := domain.RegisterEventFactory(user.EmailAddressWasChangedType, func() interface{} { return &user.EmailAddressWasChanged{} }); err != nil {
		return apperrors.Wrap(err)
	}
	if err := domain.RegisterEventFactory(user.AccessTokenWasRequestedType, func() interface{} { return &user.AccessTokenWasRequested{} }); err != nil {
		return apperrors.Wrap(err)
	}
	if err := domain.RegisterEventFactory(user.ConnectedWithGoogleType, func() interface{} { return &user.ConnectedWithGoogle{} }); err != nil {
		return apperrors.Wrap(err)
	}
	if err := domain.RegisterEventFactory(user.ConnectedWithFacebookType, func() interface{} { return &user.ConnectedWithFacebook{} }); err != nil {
		return apperrors.Wrap(err)
	}

	if err := container.CommandBus.Subscribe(ctx, user.RegisterWithEmailName, user.OnRegisterWithEmail(container.UserRepository, container.UserPersistenceRepository)); err != nil {
		return apperrors.Wrap(err)
	}
	if err := container.CommandBus.Subscribe(ctx, user.RequestAccessTokenName, user.OnRequestAccessToken(container.UserRepository)); err != nil {
		return apperrors.Wrap(err)
	}
	if err := container.CommandBus.Subscribe(ctx, user.RegisterWithGoogleName, user.OnRegisterWithGoogle(container.UserRepository, container.UserPersistenceRepository)); err != nil {
		return apperrors.Wrap(err)
	}
	if err := container.CommandBus.Subscribe(ctx, user.RegisterWithFacebookName, user.OnRegisterWithFacebook(container.UserRepository, container.UserPersistenceRepository)); err != nil {
		return apperrors.Wrap(err)
	}
	if err := container.CommandBus.Subscribe(ctx, user.ChangeEmailAddressName, user.OnChangeEmailAddress(container.UserRepository, container.UserPersistenceRepository)); err != nil {
		return apperrors.Wrap(err)
	}

	if err := container.EventBus.Subscribe(ctx, user.WasRegisteredWithEmailType, eventhandler.WhenUserWasRegisteredWithEmail(container.UserPersistenceRepository, container.CommandBus)); err != nil {
		return apperrors.Wrap(err)
	}
	if err := container.EventBus.Subscribe(ctx, user.WasRegisteredWithGoogleType, eventhandler.WhenUserWasRegisteredWithGoogle(container.UserPersistenceRepository, container.CommandBus)); err != nil {
		return apperrors.Wrap(err)
	}
	if err := container.EventBus.Subscribe(ctx, user.WasRegisteredWithFacebookType, eventhandler.WhenUserWasRegisteredWithFacebook(container.UserPersistenceRepository, container.CommandBus)); err != nil {
		return apperrors.Wrap(err)
	}
	if err := container.EventBus.Subscribe(ctx, user.EmailAddressWasChangedType, eventhandler.WhenUserEmailAddressWasChanged(container.UserPersistenceRepository)); err != nil {
		return apperrors.Wrap(err)
	}
	if err := container.EventBus.Subscribe(ctx, user.AccessTokenWasRequestedType, eventhandler.WhenUserAccessTokenWasRequested(cfg, jwt.SigningMethodHS512, container.Authenticator, container.UserPersistenceRepository, container.AuthClient)); err != nil {
		return apperrors.Wrap(err)
	}
	if err := container.EventBus.Subscribe(ctx, user.ConnectedWithGoogleType, eventhandler.WhenUserConnectedWithGoogle(container.UserPersistenceRepository, container.CommandBus)); err != nil {
		return apperrors.Wrap(err)
	}
	if err := container.EventBus.Subscribe(ctx, user.ConnectedWithFacebookType, eventhandler.WhenUserConnectedWithFacebook(container.UserPersistenceRepository, container.CommandBus)); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}
