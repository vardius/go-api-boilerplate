package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/vardius/gocontainer"
	pushpullproto "github.com/vardius/pushpull/proto"
	"google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/eventhandler"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/oauth2"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	persistence "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence/mysql"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/repository"
	usergrpc "github.com/vardius/go-api-boilerplate/cmd/user/internal/interfaces/grpc"
	userhttp "github.com/vardius/go-api-boilerplate/cmd/user/internal/interfaces/http"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	oauth2util "github.com/vardius/go-api-boilerplate/pkg/auth/oauth2"
	"github.com/vardius/go-api-boilerplate/pkg/buildinfo"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus/memory"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus/pushpull"
	eventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore/memory"
	grpcutils "github.com/vardius/go-api-boilerplate/pkg/grpc"
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
	eventStore := eventstore.New()
	oauth2Config := oauth2.NewConfig()
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
	commandBus := memory.New(config.Env.CommandBus.QueueSize, logger)

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
	grpcPushPullConn := grpcutils.NewConnection(
		ctx,
		config.Env.PushPull.Host,
		config.Env.GRPC.Port,
		grpcutils.ConnectionConfig{
			ConnTime:    config.Env.GRPC.ConnTime,
			ConnTimeout: config.Env.GRPC.ConnTimeout,
		},
		logger,
	)
	defer grpcPushPullConn.Close()
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

	grpPushPullClient := pushpullproto.NewPushPullClient(grpcPushPullConn)
	eventBus := pushpull.New(config.Env.App.EventHandlerTimeout, grpPushPullClient, logger)
	userPersistenceRepository := persistence.NewUserRepository(mysqlConnection)
	userRepository := repository.NewUserRepository(eventStore, eventBus)
	grpcHealthServer := grpchealth.NewServer()
	grpcUserServer := usergrpc.NewServer(commandBus, userPersistenceRepository, logger)
	authenticator := auth.NewSecretAuthenticator([]byte(config.Env.Auth.Secret))
	tokenProvider := oauth2util.NewCredentialsAuthenticator(config.Env.App.Secret, oauth2Config)
	claimsProvider := auth.NewClaimsProvider(authenticator)
	tokenAuthorizer := auth.NewJWTTokenAuthorizer(claimsProvider)
	router := userhttp.NewRouter(
		logger,
		tokenAuthorizer,
		userPersistenceRepository,
		commandBus,
		tokenProvider,
		mysqlConnection,
		map[string]*grpc.ClientConn{
			"pushpull": grpcPushPullConn,
			"user":     grpcUserConn,
		},
	)
	app := application.New(logger)

	commandBus.Subscribe(ctx, (user.RegisterWithEmail{}).GetName(), user.OnRegisterWithEmail(userRepository, mysqlConnection))
	commandBus.Subscribe(ctx, (user.RegisterWithGoogle{}).GetName(), user.OnRegisterWithGoogle(userRepository, mysqlConnection))
	commandBus.Subscribe(ctx, (user.RegisterWithFacebook{}).GetName(), user.OnRegisterWithFacebook(userRepository, mysqlConnection))
	commandBus.Subscribe(ctx, (user.ChangeEmailAddress{}).GetName(), user.OnChangeEmailAddress(userRepository, mysqlConnection))
	commandBus.Subscribe(ctx, (user.RequestAccessToken{}).GetName(), user.OnRequestAccessToken(userRepository, mysqlConnection))

	go func() {
		eventbus.RegisterGRPCHandlers(
			"pushpull",
			grpcPushPullConn,
			eventBus,
			map[string]eventbus.EventHandler{
				(user.WasRegisteredWithEmail{}).GetType():    eventhandler.WhenUserWasRegisteredWithEmail(mysqlConnection, userPersistenceRepository),
				(user.WasRegisteredWithGoogle{}).GetType():   eventhandler.WhenUserWasRegisteredWithGoogle(mysqlConnection, userPersistenceRepository),
				(user.WasRegisteredWithFacebook{}).GetType(): eventhandler.WhenUserWasRegisteredWithFacebook(mysqlConnection, userPersistenceRepository),
				(user.EmailAddressWasChanged{}).GetType():    eventhandler.WhenUserEmailAddressWasChanged(mysqlConnection, userPersistenceRepository),
				(user.AccessTokenWasRequested{}).GetType():   eventhandler.WhenUserAccessTokenWasRequested(tokenProvider),
			},
			5*time.Second,
		)
	}()

	app.AddAdapters(
		userhttp.NewAdapter(
			fmt.Sprintf("%s:%d", config.Env.HTTP.Host, config.Env.HTTP.Port),
			router,
		),
		usergrpc.NewAdapter(
			fmt.Sprintf("%s:%d", config.Env.GRPC.Host, config.Env.GRPC.Port),
			grpcServer,
			grpcHealthServer,
			grpcUserServer,
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
