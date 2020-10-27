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

	authproto "github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/eventhandler"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/identity"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	persistence "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence/mysql"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/repository"
	usergrpc "github.com/vardius/go-api-boilerplate/cmd/user/internal/interfaces/grpc"
	userhttp "github.com/vardius/go-api-boilerplate/cmd/user/internal/interfaces/http"
	userproto "github.com/vardius/go-api-boilerplate/cmd/user/proto"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	oauth2util "github.com/vardius/go-api-boilerplate/pkg/auth/oauth2"
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
		// @TODO: Secure grpc server with firewall
		nil, // []grpc.UnaryServerInterceptor{
		// 	firewall.GrantAccessForUnaryRequest(identity.RoleUser),
		// },
		nil, // []grpc.StreamServerInterceptor{
		// 	firewall.GrantAccessForStreamRequest(identity.RoleUser),
		// },
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
	grpcUserConn := grpcutils.NewConnection(
		ctx,
		config.Env.GRPC.Host,
		config.Env.GRPC.Port,
		grpcutils.ConnectionConfig{
			ConnTime:    config.Env.GRPC.ConnTime,
			ConnTimeout: config.Env.GRPC.ConnTimeout,
		},
		logger,
	)
	defer grpcUserConn.Close()
	grpcAuthConn := grpcutils.NewConnection(
		ctx,
		config.Env.Auth.Host,
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
	userPersistenceRepository := persistence.NewUserRepository(mysqlConnection)
	clientPersistenceRepository := persistence.NewClientRepository(mysqlConnection)
	userRepository := repository.NewUserRepository(eventStore, eventBus)
	grpcUserServer := usergrpc.NewServer(commandBus, userPersistenceRepository)
	grpAuthClient := authproto.NewAuthenticationServiceClient(grpcAuthConn)
	authenticator := auth.NewSecretAuthenticator([]byte(config.Env.Auth.Secret))
	tokenProvider := oauth2util.NewCredentialsAuthenticator(config.Env.Auth.Host, config.Env.HTTP.Port, config.Env.Auth.Secret)
	claimsProvider := auth.NewClaimsProvider(authenticator)
	identityProvider := identity.NewIdentityProvider(clientPersistenceRepository, userPersistenceRepository)
	tokenAuthorizer := auth.NewJWTTokenAuthorizer(grpAuthClient, claimsProvider, identityProvider)
	router := userhttp.NewRouter(
		logger,
		tokenAuthorizer,
		userPersistenceRepository,
		commandBus,
		tokenProvider,
		mysqlConnection,
		identityProvider,
		map[string]*grpc.ClientConn{
			"user": grpcUserConn,
		},
	)
	app := application.New(logger)

	if err := commandBus.Subscribe(ctx, (user.RegisterWithEmail{}).GetName(), user.OnRegisterWithEmail(userRepository, mysqlConnection)); err != nil {
		panic(err)
	}
	if err := commandBus.Subscribe(ctx, (user.RegisterWithGoogle{}).GetName(), user.OnRegisterWithGoogle(userRepository, mysqlConnection)); err != nil {
		panic(err)
	}
	if err := commandBus.Subscribe(ctx, (user.RegisterWithFacebook{}).GetName(), user.OnRegisterWithFacebook(userRepository, mysqlConnection)); err != nil {
		panic(err)
	}
	if err := commandBus.Subscribe(ctx, (user.ChangeEmailAddress{}).GetName(), user.OnChangeEmailAddress(userRepository, mysqlConnection)); err != nil {
		panic(err)
	}

	if err := eventBus.Subscribe(ctx, (user.WasRegisteredWithEmail{}).GetType(), eventhandler.WhenUserWasRegisteredWithEmail(mysqlConnection, userPersistenceRepository, tokenProvider, grpAuthClient)); err != nil {
		panic(err)
	}
	if err := eventBus.Subscribe(ctx, (user.WasRegisteredWithGoogle{}).GetType(), eventhandler.WhenUserWasRegisteredWithGoogle(mysqlConnection, userPersistenceRepository, grpAuthClient)); err != nil {
		panic(err)
	}
	if err := eventBus.Subscribe(ctx, (user.WasRegisteredWithFacebook{}).GetType(), eventhandler.WhenUserWasRegisteredWithFacebook(mysqlConnection, userPersistenceRepository, grpAuthClient)); err != nil {
		panic(err)
	}
	if err := eventBus.Subscribe(ctx, (user.EmailAddressWasChanged{}).GetType(), eventhandler.WhenUserEmailAddressWasChanged(mysqlConnection, userPersistenceRepository)); err != nil {
		panic(err)
	}
	if err := eventBus.Subscribe(ctx, (user.AccessTokenWasRequested{}).GetType(), eventhandler.WhenUserAccessTokenWasRequested(tokenProvider, identityProvider)); err != nil {
		panic(err)
	}
	if err := eventBus.Subscribe(ctx, (user.ConnectedWithGoogle{}).GetType(), eventhandler.WhenUserConnectedWithGoogle(mysqlConnection, userPersistenceRepository)); err != nil {
		panic(err)
	}
	if err := eventBus.Subscribe(ctx, (user.ConnectedWithFacebook{}).GetType(), eventhandler.WhenUserConnectedWithFacebook(mysqlConnection, userPersistenceRepository)); err != nil {
		panic(err)
	}

	userproto.RegisterUserServiceServer(grpcServer, grpcUserServer)

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
			"user",
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
