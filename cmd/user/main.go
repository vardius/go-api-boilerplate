package main

import (
	"context"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	auth_proto "github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	config "github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	eventhandler "github.com/vardius/go-api-boilerplate/cmd/user/internal/application/eventhandler"
	oauth2 "github.com/vardius/go-api-boilerplate/cmd/user/internal/application/oauth2"
	user "github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	persistence "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence/mysql"
	repository "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/repository"
	user_grpc "github.com/vardius/go-api-boilerplate/cmd/user/internal/interfaces/grpc"
	user_http "github.com/vardius/go-api-boilerplate/cmd/user/internal/interfaces/http"
	application "github.com/vardius/go-api-boilerplate/internal/application"
	buildinfo "github.com/vardius/go-api-boilerplate/internal/buildinfo"
	commandbus "github.com/vardius/go-api-boilerplate/internal/commandbus"
	debug "github.com/vardius/go-api-boilerplate/internal/debug"
	eventbus "github.com/vardius/go-api-boilerplate/internal/eventbus"
	eventstore "github.com/vardius/go-api-boilerplate/internal/eventstore/memory"
	grpc_utils "github.com/vardius/go-api-boilerplate/internal/grpc"
	log "github.com/vardius/go-api-boilerplate/internal/log"
	mysql "github.com/vardius/go-api-boilerplate/internal/mysql"
	pubsub_proto "github.com/vardius/pubsub/proto"
	grpc "google.golang.org/grpc"
	grpc_health "google.golang.org/grpc/health"
)

func main() {
	buildinfo.PrintVersionOrContinue()

	ctx := context.Background()

	logger := log.New(config.Env.APP.Environment)
	eventStore := eventstore.New()
	oauth2Config := oauth2.NewConfig()
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
			Host:            config.Env.DB.Host,
			Port:            config.Env.DB.Port,
			User:            config.Env.DB.User,
			Pass:            config.Env.DB.Pass,
			Database:        config.Env.DB.Database,
			ConnMaxLifetime: config.Env.DB.ConnMaxLifetime,
			MaxIdleConns:    config.Env.DB.MaxIdleConns,
			MaxOpenConns:    config.Env.DB.MaxOpenConns,
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
			"user":   grpcUserConn,
			"auth":   grpcAuthConn,
			"pubsub": grpcPubsubConn,
		},
		oauth2Config,
		config.Env.APP.Secret,
	)
	app := application.New(logger)

	commandBus.Subscribe((user.RegisterWithEmail{}).GetName(), user.OnRegisterWithEmail(userRepository, mysqlConnection))
	commandBus.Subscribe((user.RegisterWithGoogle{}).GetName(), user.OnRegisterWithGoogle(userRepository, mysqlConnection))
	commandBus.Subscribe((user.RegisterWithFacebook{}).GetName(), user.OnRegisterWithFacebook(userRepository, mysqlConnection))
	commandBus.Subscribe((user.ChangeEmailAddress{}).GetName(), user.OnChangeEmailAddress(userRepository, mysqlConnection))
	commandBus.Subscribe((user.RequestAccessToken{}).GetName(), user.OnRequestAccessToken(userRepository, mysqlConnection))

	go func() {
		eventhandler.Register(
			grpcPubsubConn,
			eventBus,
			map[string]eventbus.EventHandler{
				(user.WasRegisteredWithEmail{}).GetType():    eventhandler.WhenUserWasRegisteredWithEmail(mysqlConnection, userPersistenceRepository),
				(user.WasRegisteredWithGoogle{}).GetType():   eventhandler.WhenUserWasRegisteredWithGoogle(mysqlConnection, userPersistenceRepository),
				(user.WasRegisteredWithFacebook{}).GetType(): eventhandler.WhenUserWasRegisteredWithFacebook(mysqlConnection, userPersistenceRepository),
				(user.EmailAddressWasChanged{}).GetType():    eventhandler.WhenUserEmailAddressWasChanged(mysqlConnection, userPersistenceRepository),
				(user.AccessTokenWasRequested{}).GetType():   eventhandler.WhenUserAccessTokenWasRequested(oauth2Config, config.Env.APP.Secret),
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

	if config.Env.APP.Environment == "development" {
		app.AddAdapters(
			debug.NewAdapter(
				fmt.Sprintf("%s:%d", config.Env.Debug.Host, config.Env.Debug.Port),
			),
		)
	}

	app.WithShutdownTimeout(config.Env.APP.ShutdownTimeout)
	app.Run(ctx)
}
