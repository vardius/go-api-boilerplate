package main

import (
	"context"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	config "github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	eventhandler "github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/eventhandler"
	oauth2 "github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/oauth2"
	client "github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/client"
	token "github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	persistence "github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence/mysql"
	repository "github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/repository"
	auth_grpc "github.com/vardius/go-api-boilerplate/cmd/auth/internal/interfaces/grpc"
	auth_http "github.com/vardius/go-api-boilerplate/cmd/auth/internal/interfaces/http"
	application "github.com/vardius/go-api-boilerplate/internal/application"
	buildinfo "github.com/vardius/go-api-boilerplate/internal/buildinfo"
	commandbus "github.com/vardius/go-api-boilerplate/internal/commandbus"
	debug "github.com/vardius/go-api-boilerplate/internal/appdebug"
	eventbus "github.com/vardius/go-api-boilerplate/internal/eventbus"
	eventstore "github.com/vardius/go-api-boilerplate/internal/eventstore/memory"
	grpc_utils "github.com/vardius/go-api-boilerplate/internal/grpc"
	log "github.com/vardius/go-api-boilerplate/internal/log"
	mysql "github.com/vardius/go-api-boilerplate/internal/mysql"
	pubsub_proto "github.com/vardius/pubsub/proto"
	"google.golang.org/grpc"
	grpc_health "google.golang.org/grpc/health"
	oauth2_models "gopkg.in/oauth2.v3/models"
)

func main() {
	buildinfo.PrintVersionOrContinue()

	ctx := context.Background()

	logger := log.New(config.Env.APP.Environment)
	eventStore := eventstore.New()
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
		config.Env.GRPC.Host,
		config.Env.GRPC.Port,
		grpc_utils.ConnectionConfig{
			ConnTime:    config.Env.GRPC.ConnTime,
			ConnTimeout: config.Env.GRPC.ConnTimeout,
		},
		logger,
	)
	defer grpcAuthConn.Close()

	grpPubsubClient := pubsub_proto.NewMessageBusClient(grpcPubsubConn)
	eventBus := eventbus.New(grpPubsubClient, logger)
	tokenRepository := repository.NewTokenRepository(eventStore, eventBus)
	clientRepository := repository.NewClientRepository(eventStore, eventBus)
	tokenPersistenceRepository := persistence.NewTokenRepository(mysqlConnection)
	clientPersistenceRepository := persistence.NewClientRepository(mysqlConnection)
	tokenStore := oauth2.NewTokenStore(tokenPersistenceRepository, commandBus)
	clientStore := oauth2.NewClientStore(clientPersistenceRepository)
	manager := oauth2.NewManager(tokenStore, clientStore, []byte(config.Env.APP.Secret))
	oauth2Server := oauth2.InitServer(manager, mysqlConnection, logger, config.Env.APP.Secret)
	grpcHealthServer := grpc_health.NewServer()
	grpcAuthServer := auth_grpc.NewServer(oauth2Server, logger, config.Env.APP.Secret)
	router := auth_http.NewRouter(
		logger,
		oauth2Server,
		mysqlConnection,
		map[string]*grpc.ClientConn{
			"auth":   grpcAuthConn,
			"pubsub": grpcPubsubConn,
		},
	)
	app := application.New(logger)

	// store our internal user service client
	clientStore.SetInternal(config.Env.User.ClientID, &oauth2_models.Client{
		ID:     config.Env.User.ClientID,
		Secret: config.Env.User.ClientSecret,
		Domain: fmt.Sprintf("http://%s:%d", config.Env.User.Host, config.Env.HTTP.Port),
	})

	commandBus.Subscribe((token.Create{}).GetName(), token.OnCreate(tokenRepository, mysqlConnection))
	commandBus.Subscribe((token.Remove{}).GetName(), token.OnRemove(tokenRepository, mysqlConnection))
	commandBus.Subscribe((client.Create{}).GetName(), client.OnCreate(clientRepository, mysqlConnection))
	commandBus.Subscribe((client.Remove{}).GetName(), client.OnRemove(clientRepository, mysqlConnection))

	go func() {
		go func() {
			eventhandler.Register(
				grpcPubsubConn,
				eventBus,
				map[string]eventbus.EventHandler{
					(token.WasCreated{}).GetType():  eventhandler.WhenTokenWasCreated(mysqlConnection, tokenPersistenceRepository),
					(token.WasRemoved{}).GetType():  eventhandler.WhenTokenWasRemoved(mysqlConnection, tokenPersistenceRepository),
					(client.WasCreated{}).GetType(): eventhandler.WhenClientWasCreated(mysqlConnection, clientPersistenceRepository),
					(client.WasRemoved{}).GetType(): eventhandler.WhenClientWasRemoved(mysqlConnection, clientPersistenceRepository),
				},
				5*time.Minute,
			)
		}()
	}()

	app.AddAdapters(
		auth_http.NewAdapter(
			fmt.Sprintf("%s:%d", config.Env.HTTP.Host, config.Env.HTTP.Port),
			router,
		),
		auth_grpc.NewAdapter(
			fmt.Sprintf("%s:%d", config.Env.GRPC.Host, config.Env.GRPC.Port),
			grpcServer,
			grpcHealthServer,
			grpcAuthServer,
		),
	)

	if config.Env.APP.Environment == "development" {
		app.AddAdapters(
			application.NewDebugAdapter(
				fmt.Sprintf("%s:%d", config.Env.Debug.Host, config.Env.Debug.Port),
			),
		)
	}

	app.WithShutdownTimeout(config.Env.APP.ShutdownTimeout)
	app.Run(ctx)
}
