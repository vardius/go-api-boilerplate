package services

import (
	"context"
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/vardius/golog"
	"google.golang.org/grpc"
	"gopkg.in/oauth2.v4"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/services/identity"
	appoauth2 "github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/services/oauth2"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/client"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	authpersistence "github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	persistence "github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence/mysql"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/repository"
	authproto "github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
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
	SQL                         *sql.DB
	Logger                      golog.Logger
	CommandBus                  commandbus.CommandBus
	EventBus                    eventbus.EventBus
	AuthConn                    *grpc.ClientConn
	TokenRepository             token.Repository
	ClientRepository            client.Repository
	UserPersistenceRepository   authpersistence.UserRepository
	TokenPersistenceRepository  authpersistence.TokenRepository
	ClientPersistenceRepository authpersistence.ClientRepository
	Authenticator               auth.Authenticator
	OAuth2Manager               oauth2.Manager
	TokenAuthorizer             auth.TokenAuthorizer
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
	grpcAuthConn := grpcutils.NewConnection(
		ctx,
		cfg.GRPC.Host,
		cfg.GRPC.Port,
		grpcutils.ConnectionConfig{
			ConnTime:    cfg.GRPC.ConnTime,
			ConnTimeout: cfg.GRPC.ConnTimeout,
		},
		logger,
	)
	eventStore := mysqleventstore.New(mysqlConnection)
	eventBus := memoryeventbus.New(cfg.EventBus.QueueSize, logger)
	tokenRepository := repository.NewTokenRepository(eventStore, eventBus)
	clientRepository := repository.NewClientRepository(eventStore, eventBus)
	userPersistenceRepository := persistence.NewUserRepository(mysqlConnection)
	tokenPersistenceRepository := persistence.NewTokenRepository(mysqlConnection)
	clientPersistenceRepository := persistence.NewClientRepository(cfg, mysqlConnection)
	tokenStore := appoauth2.NewTokenStore(tokenPersistenceRepository, tokenRepository)
	authenticator := auth.NewSecretAuthenticator([]byte(cfg.App.Secret))
	manager := appoauth2.NewManager(tokenStore, clientPersistenceRepository, authenticator)
	grpAuthClient := authproto.NewAuthenticationServiceClient(grpcAuthConn)
	claimsProvider := auth.NewClaimsProvider(authenticator)
	identityProvider := identity.NewIdentityProvider(clientPersistenceRepository, userPersistenceRepository)
	tokenAuthorizer := auth.NewJWTTokenAuthorizer(grpAuthClient, claimsProvider, identityProvider)

	return &ServiceContainer{
		SQL:                         mysqlConnection,
		Logger:                      logger,
		CommandBus:                  commandBus,
		EventBus:                    eventBus,
		Authenticator:               authenticator,
		OAuth2Manager:               manager,
		AuthConn:                    grpcAuthConn,
		TokenAuthorizer:             tokenAuthorizer,
		TokenRepository:             tokenRepository,
		ClientRepository:            clientRepository,
		UserPersistenceRepository:   userPersistenceRepository,
		TokenPersistenceRepository:  tokenPersistenceRepository,
		ClientPersistenceRepository: clientPersistenceRepository,
	}, nil
}

func (c *ServiceContainer) Close() error {
	if err := c.SQL.Close(); err != nil {
		return err
	}
	if err := c.AuthConn.Close(); err != nil {
		return err
	}

	return nil
}
