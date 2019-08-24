package main

import (
	"context"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	auth_proto "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/proto"
	application "github.com/vardius/go-api-boilerplate/cmd/user/application"
	config "github.com/vardius/go-api-boilerplate/cmd/user/application/config"
	eventhandler "github.com/vardius/go-api-boilerplate/cmd/user/application/eventhandler"
	oauth2 "github.com/vardius/go-api-boilerplate/cmd/user/application/oauth2"
	user "github.com/vardius/go-api-boilerplate/cmd/user/domain/user"
	persistence "github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/persistence/mysql"
	repository "github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/repository"
	user_grpc "github.com/vardius/go-api-boilerplate/cmd/user/interfaces/grpc"
	user_http "github.com/vardius/go-api-boilerplate/cmd/user/interfaces/http"
	buildinfo "github.com/vardius/go-api-boilerplate/pkg/buildinfo"
	commandbus "github.com/vardius/go-api-boilerplate/pkg/commandbus"
	eventbus "github.com/vardius/go-api-boilerplate/pkg/eventbus"
	eventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore/memory"
	grpc_utils "github.com/vardius/go-api-boilerplate/pkg/grpc"
	log "github.com/vardius/go-api-boilerplate/pkg/log"
	mysql "github.com/vardius/go-api-boilerplate/pkg/mysql"
	pubsub_proto "github.com/vardius/pubsub/proto"
	grpc "google.golang.org/grpc"
	grpc_health "google.golang.org/grpc/health"
)

func main() {
	buildinfo.PrintVersionOrContinue()

	ctx := context.Background()

	logger := log.New(config.Env.Environment)
	eventStore := eventstore.New()
	oauth2Config := oauth2.NewConfig()
	grpcServer := grpc_utils.NewServer(config.Env, logger)
	commandBus := commandbus.New(config.Env.CommandBusQueueSize, logger)

	mysqlConnection := mysql.NewConnection(ctx, config.Env, logger)
	defer mysqlConnection.Close()
	grpcPubsubConn := grpc_utils.NewConnection(ctx, config.Env.PubSubHost, config.Env.PortGRPC, config.Env, logger)
	defer grpcPubsubConn.Close()
	grpcAuthConn := grpc_utils.NewConnection(ctx, config.Env.AuthHost, config.Env.PortGRPC, config.Env, logger)
	defer grpcAuthConn.Close()
	grpcUserConn := grpc_utils.NewConnection(ctx, config.Env.Host, config.Env.PortGRPC, config.Env, logger)
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
		config.Env.Secret,
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
				(user.AccessTokenWasRequested{}).GetType():   eventhandler.WhenUserAccessTokenWasRequested(oauth2Config, config.Env.Secret),
			},
			5*time.Minute,
		)
	}()

	app.AddAdapters(
		user_http.NewAdapter(
			fmt.Sprintf("%s:%d", config.Env.Host, config.Env.PortHTTP),
			router,
		),
		user_grpc.NewAdapter(
			fmt.Sprintf("%s:%d", config.Env.Host, config.Env.PortGRPC),
			grpcServer,
			grpcHealthServer,
			grpcUserServer,
		),
	)

	app.Run(ctx)
}
