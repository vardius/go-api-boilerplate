// +build persistence_mongodb

package services

import (
	"context"
	"fmt"
	authproto "github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	persistence "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence/mongo"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/repository"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	memorycommandbus "github.com/vardius/go-api-boilerplate/pkg/commandbus/memory"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	memoryeventbus "github.com/vardius/go-api-boilerplate/pkg/eventbus/memory"
	mongoeventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore/mongo"
	grpcutils "github.com/vardius/go-api-boilerplate/pkg/grpc"
	"github.com/vardius/go-api-boilerplate/pkg/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func init() {
	NewServiceContainer = newMongoServiceContainer
}

func newMongoServiceContainer(ctx context.Context, cfg *config.Config) (*ServiceContainer, error) {
	logger := log.New(cfg.App.Environment)
	commandBus := memorycommandbus.New(cfg.CommandBus.QueueSize, logger)
	mongoConnection, err := mongo.Connect(ctx, options.Client().ApplyURI(
		fmt.Sprintf("mongodb://%s:%s@%s:%d", cfg.MongoDB.User, cfg.MongoDB.Pass, cfg.MongoDB.Host, cfg.MongoDB.Port),
	))
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	mongoDB := mongoConnection.Database(cfg.MongoDB.Database)
	grpcUserConn := grpcutils.NewConnection(
		ctx,
		cfg.GRPC.Host,
		cfg.GRPC.Port,
		grpcutils.ConnectionConfig{
			ConnTime:    cfg.GRPC.ConnTime,
			ConnTimeout: cfg.GRPC.ConnTimeout,
		},
		logger,
	)
	grpcAuthConn := grpcutils.NewConnection(
		ctx,
		cfg.Auth.Host,
		cfg.GRPC.Port,
		grpcutils.ConnectionConfig{
			ConnTime:    cfg.GRPC.ConnTime,
			ConnTimeout: cfg.GRPC.ConnTimeout,
		},
		logger,
	)
	eventStore, err := mongoeventstore.New(ctx, "events", mongoDB)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	eventBus := memoryeventbus.New(cfg.EventBus.QueueSize, logger)
	userPersistenceRepository, err := persistence.NewUserRepository(ctx, mongoDB)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	userRepository := repository.NewUserRepository(eventStore, eventBus)
	grpAuthClient := authproto.NewAuthenticationServiceClient(grpcAuthConn)
	authenticator := auth.NewSecretAuthenticator([]byte(cfg.Auth.Secret))
	claimsProvider := auth.NewClaimsProvider(authenticator)
	tokenAuthorizer := auth.NewJWTTokenAuthorizer(grpAuthClient, claimsProvider, authenticator)

	return &ServiceContainer{
		Logger:                    logger,
		Mongo:                     mongoConnection,
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
