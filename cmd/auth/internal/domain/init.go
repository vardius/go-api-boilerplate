package domain

import (
	"context"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/eventhandler"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/services"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/client"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
)

func RegisterTokenDomain(ctx context.Context, cfg *config.Config, container *services.ServiceContainer) error {
	if err := domain.RegisterEventFactory(token.WasCreatedType, func() interface{} { return &token.WasCreated{} }); err != nil {
		return apperrors.Wrap(err)
	}
	if err := domain.RegisterEventFactory(token.WasRemovedType, func() interface{} { return &token.WasRemoved{} }); err != nil {
		return apperrors.Wrap(err)
	}

	if err := container.CommandBus.Subscribe(ctx, token.CreateName, token.OnCreate(container.TokenRepository)); err != nil {
		return apperrors.Wrap(err)
	}
	if err := container.CommandBus.Subscribe(ctx, token.RemoveName, token.OnRemove(container.TokenRepository)); err != nil {
		return apperrors.Wrap(err)
	}

	if err := container.EventBus.Subscribe(ctx, token.WasCreatedType, eventhandler.WhenTokenWasCreated(container.TokenPersistenceRepository)); err != nil {
		return apperrors.Wrap(err)
	}
	if err := container.EventBus.Subscribe(ctx, token.WasRemovedType, eventhandler.WhenTokenWasRemoved(container.TokenPersistenceRepository)); err != nil {
		return apperrors.Wrap(err)
	}
	return nil
}

func RegisterClientDomain(ctx context.Context, cfg *config.Config, container *services.ServiceContainer) error {
	if err := domain.RegisterEventFactory(client.WasCreatedType, func() interface{} { return &client.WasCreated{} }); err != nil {
		return apperrors.Wrap(err)
	}
	if err := domain.RegisterEventFactory(client.WasRemovedType, func() interface{} { return &client.WasRemoved{} }); err != nil {
		return apperrors.Wrap(err)
	}

	if err := container.CommandBus.Subscribe(ctx, client.CreateName, client.OnCreate(container.ClientRepository)); err != nil {
		return apperrors.Wrap(err)
	}
	if err := container.CommandBus.Subscribe(ctx, client.RemoveName, client.OnRemove(container.ClientRepository)); err != nil {
		return apperrors.Wrap(err)
	}

	if err := container.EventBus.Subscribe(ctx, client.WasCreatedType, eventhandler.WhenClientWasCreated(container.ClientPersistenceRepository)); err != nil {
		return apperrors.Wrap(err)
	}
	if err := container.EventBus.Subscribe(ctx, client.WasRemovedType, eventhandler.WhenClientWasRemoved(container.ClientPersistenceRepository)); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}
