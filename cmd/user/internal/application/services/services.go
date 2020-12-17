package services

import (
	"context"
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/vardius/golog"
	"google.golang.org/grpc"

	authproto "github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/services/identity"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	userpersistence "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	persistence "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence/mysql"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/repository"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	oauth2util "github.com/vardius/go-api-boilerplate/pkg/auth/oauth2"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	memorycommandbus "github.com/vardius/go-api-boilerplate/pkg/commandbus/memory"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	memoryeventbus "github.com/vardius/go-api-boilerplate/pkg/eventbus/memory"
	mysqleventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore/mysql"
	grpcutils "github.com/vardius/go-api-boilerplate/pkg/grpc"
	"github.com/vardius/go-api-boilerplate/pkg/log"
	"github.com/vardius/go-api-boilerplate/pkg/mysql"
)

type ServiceContainer struct {
	SQL                       *sql.DB
	Logger                    golog.Logger
	CommandBus                commandbus.CommandBus
	EventBus                  eventbus.EventBus
	UserConn                  *grpc.ClientConn
	AuthConn                  *grpc.ClientConn
	UserRepository            user.Repository
	UserPersistenceRepository userpersistence.UserRepository
	AuthClient                authproto.AuthenticationServiceClient
	TokenProvider             oauth2util.TokenProvider
	IdentityProvider          identity.Provider
	TokenAuthorizer           auth.TokenAuthorizer
}

func NewServiceContainer(ctx context.Context, cfg *config.Config) (*ServiceContainer, error) {
	logger := log.New(cfg.App.Environment)
	commandBus := memorycommandbus.New(cfg.CommandBus.QueueSize, logger)
	mysqlConnection := mysql.NewConnection(
		ctx,
		mysql.ConnectionConfig{
			Host:            cfg.MYSQL.Host,
			Port:            cfg.MYSQL.Port,
			User:            cfg.MYSQL.User,
			Pass:            cfg.MYSQL.Pass,
			Database:        cfg.MYSQL.Database,
			ConnMaxLifetime: cfg.MYSQL.ConnMaxLifetime,
			MaxIdleConns:    cfg.MYSQL.MaxIdleConns,
			MaxOpenConns:    cfg.MYSQL.MaxOpenConns,
		},
		logger,
	)
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
	eventStore := mysqleventstore.New(mysqlConnection)
	eventBus := memoryeventbus.New(cfg.EventBus.QueueSize, logger)
	userPersistenceRepository := persistence.NewUserRepository(mysqlConnection)
	clientPersistenceRepository := persistence.NewClientRepository(mysqlConnection)
	userRepository := repository.NewUserRepository(eventStore, eventBus)
	grpAuthClient := authproto.NewAuthenticationServiceClient(grpcAuthConn)
	authenticator := auth.NewSecretAuthenticator([]byte(cfg.Auth.Secret))
	tokenProvider := oauth2util.NewCredentialsAuthenticator(cfg.Auth.Host, cfg.HTTP.Port, cfg.Auth.Secret)
	claimsProvider := auth.NewClaimsProvider(authenticator)
	identityProvider := identity.NewIdentityProvider(cfg, clientPersistenceRepository, userPersistenceRepository)
	tokenAuthorizer := auth.NewJWTTokenAuthorizer(grpAuthClient, claimsProvider, identityProvider)

	return &ServiceContainer{
		Logger:                    logger,
		SQL:                       mysqlConnection,
		CommandBus:                commandBus,
		UserConn:                  grpcUserConn,
		AuthConn:                  grpcAuthConn,
		EventBus:                  eventBus,
		AuthClient:                grpAuthClient,
		TokenProvider:             tokenProvider,
		IdentityProvider:          identityProvider,
		TokenAuthorizer:           tokenAuthorizer,
		UserRepository:            userRepository,
		UserPersistenceRepository: userPersistenceRepository,
	}, nil
}

func (c *ServiceContainer) Close() error {
	if err := c.SQL.Close(); err != nil {
		return err
	}
	if err := c.UserConn.Close(); err != nil {
		return err
	}
	if err := c.AuthConn.Close(); err != nil {
		return err
	}

	return nil
}
