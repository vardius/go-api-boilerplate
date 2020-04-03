package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	_ "github.com/go-sql-driver/mysql"
	pubsub_proto "github.com/vardius/pubsub/v2/proto"
	pushpull_proto "github.com/vardius/pushpull/proto"
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
	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/buildinfo"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	eventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore/memory"
	grpc_utils "github.com/vardius/go-api-boilerplate/pkg/grpc"
	"github.com/vardius/go-api-boilerplate/pkg/log"
	"github.com/vardius/go-api-boilerplate/pkg/mysql"
)

func main() {
	buildinfo.PrintVersionOrContinue()

	rand.Seed(time.Now().UnixNano())

	ctx := context.Background()

	logger := log.New(config.Env.App.Environment)
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
	grpcPubSubConn := grpc_utils.NewConnection(
		ctx,
		config.Env.PubSub.Host,
		config.Env.GRPC.Port,
		grpc_utils.ConnectionConfig{
			ConnTime:    config.Env.GRPC.ConnTime,
			ConnTimeout: config.Env.GRPC.ConnTimeout,
		},
		logger,
	)
	defer grpcPubSubConn.Close()
	grpcPushPullConn := grpc_utils.NewConnection(
		ctx,
		config.Env.PushPull.Host,
		config.Env.GRPC.Port,
		grpc_utils.ConnectionConfig{
			ConnTime:    config.Env.GRPC.ConnTime,
			ConnTimeout: config.Env.GRPC.ConnTimeout,
		},
		logger,
	)
	defer grpcPushPullConn.Close()
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

	grpcPubsubClient := pubsub_proto.NewPubSubClient(grpcPubSubConn)
	grpPushPullClient := pushpull_proto.NewPushPullClient(grpcPushPullConn)
	eventBus := eventbus.New(grpcPubsubClient, grpPushPullClient, logger)
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
			"auth":     grpcAuthConn,
			"pushpull": grpcPushPullConn,
			"pubsub":   grpcPubSubConn,
			"user":     grpcUserConn,
		},
		oauth2Config,
		config.Env.App.Secret,
	)
	app := application.New(logger)

	commandBus.Subscribe((user.RegisterWithEmail{}).GetName(), user.OnRegisterWithEmail(userRepository, mysqlConnection))
	commandBus.Subscribe((user.RegisterWithGoogle{}).GetName(), user.OnRegisterWithGoogle(userRepository, mysqlConnection))
	commandBus.Subscribe((user.RegisterWithFacebook{}).GetName(), user.OnRegisterWithFacebook(userRepository, mysqlConnection))
	commandBus.Subscribe((user.ChangeEmailAddress{}).GetName(), user.OnChangeEmailAddress(userRepository, mysqlConnection))
	commandBus.Subscribe((user.RequestAccessToken{}).GetName(), user.OnRequestAccessToken(userRepository, mysqlConnection))

	go func() {
		eventbus.RegisterHandlers(
			grpcPubSubConn,
			grpcPushPullConn,
			eventBus,
			map[string]eventbus.EventHandler{
				(user.WasRegisteredWithEmail{}).GetType():    eventhandler.WhenUserWasRegisteredWithEmail(mysqlConnection, userPersistenceRepository),
				(user.WasRegisteredWithGoogle{}).GetType():   eventhandler.WhenUserWasRegisteredWithGoogle(mysqlConnection, userPersistenceRepository),
				(user.WasRegisteredWithFacebook{}).GetType(): eventhandler.WhenUserWasRegisteredWithFacebook(mysqlConnection, userPersistenceRepository),
				(user.EmailAddressWasChanged{}).GetType():    eventhandler.WhenUserEmailAddressWasChanged(mysqlConnection, userPersistenceRepository),
				(user.AccessTokenWasRequested{}).GetType():   eventhandler.WhenUserAccessTokenWasRequested(oauth2Config, config.Env.App.Secret),
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
