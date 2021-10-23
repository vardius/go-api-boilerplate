//go:build !persistence_mysql
// +build !persistence_mysql

package services

import (
	"context"

	authproto "github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	persistence "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence/memory"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/repository"
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
	grpcUserConn := grpcutils.NewConnection(
		ctx,
		cfg.GRPC.Host,
		cfg.GRPC.Port,
		grpcutils.ConnectionConfig{
			ConnTime:    cfg.GRPC.ConnTime,
			ConnTimeout: cfg.GRPC.ConnTimeout,
		},
	)
	grpcAuthConn := grpcutils.NewConnection(
		ctx,
		cfg.Auth.Host,
		cfg.GRPC.Port,
		grpcutils.ConnectionConfig{
			ConnTime:    cfg.GRPC.ConnTime,
			ConnTimeout: cfg.GRPC.ConnTimeout,
		},
	)
	eventStore := memoryeventstore.New()
	eventBus := memoryeventbus.New(cfg.EventBus.QueueSize)
	userPersistenceRepository := persistence.NewUserRepository()
	userRepository := repository.NewUserRepository(eventStore, eventBus)
	grpAuthClient := authproto.NewAuthenticationServiceClient(grpcAuthConn)
	authenticator := auth.NewSecretAuthenticator([]byte(cfg.Auth.Secret))
	claimsProvider := auth.NewClaimsProvider(authenticator)
	tokenAuthorizer := auth.NewJWTTokenAuthorizer(grpAuthClient, claimsProvider, authenticator)

	return &ServiceContainer{
		CommandBus:                commandBus,
		UserConn:                  grpcUserConn,
		AuthConn:                  grpcAuthConn,
		EventBus:                  eventBus,
		AuthClient:                grpAuthClient,
		TokenAuthorizer:           tokenAuthorizer,
		UserRepository:            userRepository,
		UserPersistenceRepository: userPersistenceRepository,
		Authenticator:             authenticator,
	}, nil
}
