//go:build persistence_mysql
// +build persistence_mysql

package services

import (
	"context"

	_ "github.com/go-sql-driver/mysql"
	authproto "github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	persistence "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence/mysql"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/repository"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	memorycommandbus "github.com/vardius/go-api-boilerplate/pkg/commandbus/memory"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	memoryeventbus "github.com/vardius/go-api-boilerplate/pkg/eventbus/memory"
	mysqleventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore/mysql"
	grpcutils "github.com/vardius/go-api-boilerplate/pkg/grpc"
	"github.com/vardius/go-api-boilerplate/pkg/mysql"
)

func init() {
	NewServiceContainer = newMYSQLServiceContainer
}

func newMYSQLServiceContainer(ctx context.Context, cfg *config.Config) (*ServiceContainer, error) {
	commandBus := memorycommandbus.New(cfg.CommandBus.QueueSize)
	sqlConn := mysql.NewConnection(
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
	)
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
	eventStore, err := mysqleventstore.New(ctx, "user_events", sqlConn)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	eventBus := memoryeventbus.New(cfg.EventBus.QueueSize)
	userPersistenceRepository, err := persistence.NewUserRepository(ctx, sqlConn)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	userRepository := repository.NewUserRepository(eventStore, eventBus)
	grpAuthClient := authproto.NewAuthenticationServiceClient(grpcAuthConn)
	authenticator := auth.NewSecretAuthenticator([]byte(cfg.Auth.Secret))
	claimsProvider := auth.NewClaimsProvider(authenticator)
	tokenAuthorizer := auth.NewJWTTokenAuthorizer(grpAuthClient, claimsProvider, authenticator)

	return &ServiceContainer{
		SQL:                       sqlConn,
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
