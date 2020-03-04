package main

import (
	"context"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/facebook"
	"github.com/markbates/goth/providers/google"
	pubsub_proto "github.com/vardius/pubsub/proto"
	"google.golang.org/grpc"
	grpc_health "google.golang.org/grpc/health"

	auth_proto "github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/eventhandler"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/oauth2"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	persistence "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence/mysql"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/repository"
	user_grpc "github.com/vardius/go-api-boilerplate/cmd/user/internal/interfaces/grpc"
	user_http "github.com/vardius/go-api-boilerplate/cmd/user/internal/interfaces/http"
	"github.com/vardius/go-api-boilerplate/internal/application"
	"github.com/vardius/go-api-boilerplate/internal/buildinfo"
	"github.com/vardius/go-api-boilerplate/internal/commandbus"
	"github.com/vardius/go-api-boilerplate/internal/eventbus"
	eventstore "github.com/vardius/go-api-boilerplate/internal/eventstore/memory"
	grpc_utils "github.com/vardius/go-api-boilerplate/internal/grpc"
	"github.com/vardius/go-api-boilerplate/internal/log"
	"github.com/vardius/go-api-boilerplate/internal/mysql"
)

func main() {
	buildinfo.PrintVersionOrContinue()

	ctx := context.Background()

	logger := log.New(config.Env.App.Environment)
	eventStore := eventstore.New()
	oauth2Config := oauth2.NewConfig()
	oauth2FacebookConfig := oauth2.NewConfigFacebook()
	oauth2GoogleConfig := oauth2.NewConfigGoogle()
	goth.UseProviders(
		facebook.New(oauth2FacebookConfig.ClientID, oauth2FacebookConfig.ClientSecret, oauth2FacebookConfig.RedirectURL),
		google.New(oauth2GoogleConfig.ClientID, oauth2GoogleConfig.ClientSecret, oauth2GoogleConfig.RedirectURL),
	)
	grpcServer := grpc_utils.NewServer(
		grpc_utils.ServerConfig{
			ServerMinTime: config.Env.GRPC.ServerMinTime,
			ServerTime:    config.Env.GRPC.ServerTime,
			ServerTimeout: config.Env.GRPC.ServerTimeout,
		},
		logger,
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
	grpcPubsubConn := grpc_utils.NewConnection(
		ctx,
		config.Env.PubSub.Host,
		config.Env.GRPC.Port,
		grpc_utils.ConnectionConfig{
			ConnTime:    config.Env.GRPC.ConnTime,
			ConnTimeout: config.Env.GRPC.ConnTimeout,
		},
		logger,
	)
	defer grpcPubsubConn.Close()
	grpcAuthConn := grpc_utils.NewConnection(
		ctx,
		config.Env.Auth.Host,
		config.Env.GRPC.Port,
		grpc_utils.ConnectionConfig{
			ConnTime:    config.Env.GRPC.ConnTime,
			ConnTimeout: config.Env.GRPC.ConnTimeout,
		},
		logger,
	)
	defer grpcAuthConn.Close()
	grpcUserConn := grpc_utils.NewConnection(
		ctx,
		config.Env.GRPC.Host,
		config.Env.GRPC.Port,
		grpc_utils.ConnectionConfig{
			ConnTime:    config.Env.GRPC.ConnTime,
			ConnTimeout: config.Env.GRPC.ConnTimeout,
		},
		logger,
	)
	defer grpcUserConn.Close()

	grpcPubsubClient := pubsub_proto.NewMessageBusClient(grpcPubsubConn)
	eventBus := eventbus.New(grpcPubsubClient, logger)
	userPersistenceRepository := persistence.NewUserRepository(mysqlConnection)
	userRepository := repository.NewUserRepository(eventStore, eventBus)
	grpcAuthClient := auth_proto.NewAuthenticationServiceClient(grpcAuthConn)
	grpcHealthServer := grpc_health.NewServer()
	grpcUserServer := user_grpc.NewServer(commandBus, userPersistenceRepository, logger)
	router := user_http.NewRouter(
		logger,
		userPersistenceRepository,
		commandBus,
		mysqlConnection,
		grpcAuthClient,
		map[string]*grpc.ClientConn{
			"auth":   grpcAuthConn,
			"pubsub": grpcPubsubConn,
			"user":   grpcUserConn,
		},
		oauth2Config,
		config.Env.App.Secret,
	)
	app := application.New(logger)

	commandBus.Subscribe((user.RegisterWithEmail{}).GetName(), user.OnRegisterWithEmail(userRepository, mysqlConnection))
	commandBus.Subscribe((user.AuthWithProvider{}).GetName(), user.OnAuthWithProvider(userRepository, mysqlConnection))
	commandBus.Subscribe((user.ChangeEmailAddress{}).GetName(), user.OnChangeEmailAddress(userRepository, mysqlConnection))
	commandBus.Subscribe((user.RequestAccessToken{}).GetName(), user.OnRequestAccessToken(userRepository, mysqlConnection))

	go func() {
		eventhandler.Register(
			grpcPubsubConn,
			eventBus,
			map[string]eventbus.EventHandler{
				(user.WasRegisteredWithEmail{}).GetType():       eventhandler.WhenUserWasRegisteredWithEmail(mysqlConnection, userPersistenceRepository),
				(user.WasAuthenticatedWithProvider{}).GetType(): eventhandler.WhenUserWasAuthenticatedWithProvider(mysqlConnection, userPersistenceRepository),
				(user.EmailAddressWasChanged{}).GetType():       eventhandler.WhenUserEmailAddressWasChanged(mysqlConnection, userPersistenceRepository),
				(user.AccessTokenWasRequested{}).GetType():      eventhandler.WhenUserAccessTokenWasRequested(oauth2Config, config.Env.App.Secret),
			},
			5*time.Minute,
		)
	}()

	app.AddAdapters(
		user_http.NewAdapter(
			fmt.Sprintf("%s:%d", config.Env.HTTP.Host, config.Env.HTTP.Port),
			router,
		),
		user_grpc.NewAdapter(
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
