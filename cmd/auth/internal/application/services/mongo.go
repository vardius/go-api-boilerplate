//go:build persistence_mongodb
// +build persistence_mongodb

package services

import (
	"context"
	"fmt"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	appoauth2 "github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/services/oauth2"
	persistence "github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence/mongo"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/repository"
	authproto "github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	memorycommandbus "github.com/vardius/go-api-boilerplate/pkg/commandbus/memory"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	memoryeventbus "github.com/vardius/go-api-boilerplate/pkg/eventbus/memory"
	mongoeventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore/mongo"
	grpcutils "github.com/vardius/go-api-boilerplate/pkg/grpc"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func init() {
	NewServiceContainer = newMYSQLServiceContainer
}

func newMYSQLServiceContainer(ctx context.Context, cfg *config.Config) (*ServiceContainer, error) {
	commandBus := memorycommandbus.New(cfg.CommandBus.QueueSize)
	mongoConnection, err := mongo.Connect(ctx, options.Client().ApplyURI(
		fmt.Sprintf("mongodb://%s:%s@%s:%d", cfg.MongoDB.User, cfg.MongoDB.Pass, cfg.MongoDB.Host, cfg.MongoDB.Port),
	))
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	mongoDB := mongoConnection.Database(cfg.MongoDB.Database)
	grpcAuthConn := grpcutils.NewConnection(
		ctx,
		cfg.GRPC.Host,
		cfg.GRPC.Port,
		grpcutils.ConnectionConfig{
			ConnTime:    cfg.GRPC.ConnTime,
			ConnTimeout: cfg.GRPC.ConnTimeout,
		},
	)
	eventStore, err := mongoeventstore.New(ctx, "events", mongoDB)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	eventBus := memoryeventbus.New(cfg.EventBus.QueueSize)
	tokenRepository := repository.NewTokenRepository(eventStore, eventBus)
	clientRepository := repository.NewClientRepository(eventStore, eventBus)
	tokenPersistenceRepository, err := persistence.NewTokenRepository(ctx, mongoDB)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	clientPersistenceRepository, err := persistence.NewClientRepository(ctx, cfg, mongoDB)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	tokenStore := appoauth2.NewTokenStore(tokenPersistenceRepository, tokenRepository)
	authenticator := auth.NewSecretAuthenticator([]byte(cfg.App.Secret))
	grpAuthClient := authproto.NewAuthenticationServiceClient(grpcAuthConn)
	claimsProvider := auth.NewClaimsProvider(authenticator)
	manager := appoauth2.NewManager(tokenStore, clientPersistenceRepository, authenticator, clientPersistenceRepository)
	tokenAuthorizer := auth.NewJWTTokenAuthorizer(grpAuthClient, claimsProvider, authenticator)

	return &ServiceContainer{
		Mongo:                       mongoConnection,
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
