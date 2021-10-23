//go:build !persistence_mysql
// +build !persistence_mysql

package services

import (
	"context"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	appoauth2 "github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/services/oauth2"
	persistence "github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence/memory"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/repository"
	authproto "github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	memorycommandbus "github.com/vardius/go-api-boilerplate/pkg/commandbus/memory"
	memoryeventbus "github.com/vardius/go-api-boilerplate/pkg/eventbus/memory"
	memoryeventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore/memory"
	grpcutils "github.com/vardius/go-api-boilerplate/pkg/grpc"
)

func init() {
	NewServiceContainer = newMemoryServiceContainer
}

func newMemoryServiceContainer(ctx context.Context, cfg *config.Config) (*ServiceContainer, error) {
	commandBus := memorycommandbus.New(cfg.CommandBus.QueueSize)
	grpcAuthConn := grpcutils.NewConnection(
		ctx,
		cfg.GRPC.Host,
		cfg.GRPC.Port,
		grpcutils.ConnectionConfig{
			ConnTime:    cfg.GRPC.ConnTime,
			ConnTimeout: cfg.GRPC.ConnTimeout,
		},
	)
	eventStore := memoryeventstore.New()
	eventBus := memoryeventbus.New(cfg.EventBus.QueueSize)
	tokenRepository := repository.NewTokenRepository(eventStore, eventBus)
	clientRepository := repository.NewClientRepository(eventStore, eventBus)
	tokenPersistenceRepository := persistence.NewTokenRepository()
	clientPersistenceRepository := persistence.NewClientRepository(cfg)
	tokenStore := appoauth2.NewTokenStore(tokenPersistenceRepository, tokenRepository)
	authenticator := auth.NewSecretAuthenticator([]byte(cfg.App.Secret))
	grpAuthClient := authproto.NewAuthenticationServiceClient(grpcAuthConn)
	claimsProvider := auth.NewClaimsProvider(authenticator)
	manager := appoauth2.NewManager(tokenStore, clientPersistenceRepository, authenticator, clientPersistenceRepository)
	tokenAuthorizer := auth.NewJWTTokenAuthorizer(grpAuthClient, claimsProvider, authenticator)

	return &ServiceContainer{
		CommandBus:                  commandBus,
		EventBus:                    eventBus,
		Authenticator:               authenticator,
		OAuth2Manager:               manager,
		AuthConn:                    grpcAuthConn,
		TokenAuthorizer:             tokenAuthorizer,
		TokenRepository:             tokenRepository,
		ClientRepository:            clientRepository,
		TokenPersistenceRepository:  tokenPersistenceRepository,
		ClientPersistenceRepository: clientPersistenceRepository,
	}, nil
}
