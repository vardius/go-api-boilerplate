package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/vardius/gocontainer"
	"google.golang.org/grpc"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/eventhandler"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/identity"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/oauth2"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/client"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	persistence "github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence/mysql"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/repository"
	authgrpc "github.com/vardius/go-api-boilerplate/cmd/auth/internal/interfaces/grpc"
	authhttp "github.com/vardius/go-api-boilerplate/cmd/auth/internal/interfaces/http"
	authproto "github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	"github.com/vardius/go-api-boilerplate/pkg/buildinfo"
	commandbus "github.com/vardius/go-api-boilerplate/pkg/commandbus/memory"
	eventbus "github.com/vardius/go-api-boilerplate/pkg/eventbus/memory"
	eventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore/mysql"
	grpcutils "github.com/vardius/go-api-boilerplate/pkg/grpc"
	httputils "github.com/vardius/go-api-boilerplate/pkg/http"
	"github.com/vardius/go-api-boilerplate/pkg/log"
	"github.com/vardius/go-api-boilerplate/pkg/mysql"
)

func init() {
	rand.Seed(time.Now().UnixNano())

	gocontainer.GlobalContainer = nil // disable global container instance
}

func main() {
	buildinfo.PrintVersionOrContinue()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := log.New(config.Env.App.Environment)
	grpcServer := grpcutils.NewServer(
		grpcutils.ServerConfig{
			ServerMinTime: config.Env.GRPC.ServerMinTime,
			ServerTime:    config.Env.GRPC.ServerTime,
			ServerTimeout: config.Env.GRPC.ServerTimeout,
		},
		logger,
		nil,
		nil,
	)
	commandBus := commandbus.New(config.Env.CommandBus.QueueSize, logger)

	mysqlConnection := mysql.NewConnection(
		ctx,
		mysql.ConnectionConfig{
			Host:            config.Env.MYSQL.Host,
			Port:            config.Env.MYSQL.Port,
			User:            config.Env.MYSQL.User,
			Pass:            config.Env.MYSQL.Pass,
			Database:        config.Env.MYSQL.Database,
			ConnMaxLifetime: config.Env.MYSQL.ConnMaxLifetime,
			MaxIdleConns:    config.Env.MYSQL.MaxIdleConns,
			MaxOpenConns:    config.Env.MYSQL.MaxOpenConns,
		},
		logger,
	)
	defer mysqlConnection.Close()
	grpcAuthConn := grpcutils.NewConnection(
		ctx,
		config.Env.GRPC.Host,
		config.Env.GRPC.Port,
		grpcutils.ConnectionConfig{
			ConnTime:    config.Env.GRPC.ConnTime,
			ConnTimeout: config.Env.GRPC.ConnTimeout,
		},
		logger,
	)
	defer grpcAuthConn.Close()

	eventStore := eventstore.New(mysqlConnection)
	eventBus := eventbus.New(config.Env.EventBus.QueueSize, logger)
	tokenRepository := repository.NewTokenRepository(eventStore, eventBus)
	clientRepository := repository.NewClientRepository(eventStore, eventBus)
	userPersistenceRepository := persistence.NewUserRepository(mysqlConnection)
	tokenPersistenceRepository := persistence.NewTokenRepository(mysqlConnection)
	clientPersistenceRepository := persistence.NewClientRepository(mysqlConnection)
	tokenStore := oauth2.NewTokenStore(tokenPersistenceRepository, tokenRepository)
	authenticator := auth.NewSecretAuthenticator([]byte(config.Env.App.Secret))
	manager := oauth2.NewManager(tokenStore, clientPersistenceRepository, authenticator)
	oauth2Server := oauth2.InitServer(manager, logger, userPersistenceRepository, clientPersistenceRepository, config.Env.App.Secret, config.Env.OAuth.InitTimeout)
	grpcAuthServer := authgrpc.NewServer(oauth2Server, clientRepository, authenticator)
	grpAuthClient := authproto.NewAuthenticationServiceClient(grpcAuthConn)
	claimsProvider := auth.NewClaimsProvider(authenticator)
	identityProvider := identity.NewIdentityProvider(clientPersistenceRepository, userPersistenceRepository)
	tokenAuthorizer := auth.NewJWTTokenAuthorizer(grpAuthClient, claimsProvider, identityProvider)
	router := authhttp.NewRouter(
		logger,
		tokenAuthorizer,
		oauth2Server,
		commandBus,
		mysqlConnection,
		map[string]*grpc.ClientConn{
			"auth": grpcAuthConn,
		},
		tokenPersistenceRepository,
		clientPersistenceRepository,
	)
	app := application.New(logger)

	// if err := commandBus.Subscribe(ctx, (token.Create{}).GetName(), token.OnCreate(tokenRepository)); err != nil {
	// 	panic(err)
	// }
	if err := commandBus.Subscribe(ctx, (token.Remove{}).GetName(), token.OnRemove(tokenRepository)); err != nil {
		panic(err)
	}
	if err := commandBus.Subscribe(ctx, (client.Create{}).GetName(), client.OnCreate(clientRepository)); err != nil {
		panic(err)
	}
	if err := commandBus.Subscribe(ctx, (client.Remove{}).GetName(), client.OnRemove(clientRepository)); err != nil {
		panic(err)
	}

	if err := eventBus.Subscribe(ctx, (token.WasCreated{}).GetType(), eventhandler.WhenTokenWasCreated(mysqlConnection, tokenPersistenceRepository)); err != nil {
		panic(err)
	}
	if err := eventBus.Subscribe(ctx, (token.WasRemoved{}).GetType(), eventhandler.WhenTokenWasRemoved(mysqlConnection, tokenPersistenceRepository)); err != nil {
		panic(err)
	}
	if err := eventBus.Subscribe(ctx, (client.WasCreated{}).GetType(), eventhandler.WhenClientWasCreated(mysqlConnection, clientPersistenceRepository)); err != nil {
		panic(err)
	}
	if err := eventBus.Subscribe(ctx, (client.WasRemoved{}).GetType(), eventhandler.WhenClientWasRemoved(mysqlConnection, clientPersistenceRepository)); err != nil {
		panic(err)
	}

	authproto.RegisterAuthenticationServiceServer(grpcServer, grpcAuthServer)

	app.AddAdapters(
		httputils.NewAdapter(
			&http.Server{
				Addr:         fmt.Sprintf("%s:%d", config.Env.HTTP.Host, config.Env.HTTP.Port),
				ReadTimeout:  config.Env.HTTP.ReadTimeout,
				WriteTimeout: config.Env.HTTP.WriteTimeout,
				IdleTimeout:  config.Env.HTTP.IdleTimeout, // limits server-side the amount of time a Keep-Alive connection will be kept idle before being reused
				Handler:      router,
			},
		),
		grpcutils.NewAdapter(
			"auth",
			fmt.Sprintf("%s:%d", config.Env.GRPC.Host, config.Env.GRPC.Port),
			grpcServer,
		),
	)

	if config.Env.App.Environment == "development" {
		app.AddAdapters(
			application.NewDebugAdapter(
				fmt.Sprintf("%s:%d", config.Env.Debug.Host, config.Env.Debug.Port),
			),
		)
	}

	app.WithShutdownTimeout(config.Env.App.ShutdownTimeout)
	app.Run(ctx)
}
