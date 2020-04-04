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
	oauth2_models "gopkg.in/oauth2.v3/models"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/eventhandler"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/oauth2"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/client"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	persistence "github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence/mysql"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/repository"
	auth_grpc "github.com/vardius/go-api-boilerplate/cmd/auth/internal/interfaces/grpc"
	auth_http "github.com/vardius/go-api-boilerplate/cmd/auth/internal/interfaces/http"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/buildinfo"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	eventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore/memory"
	grpc_utils "github.com/vardius/go-api-boilerplate/pkg/grpc"
	"github.com/vardius/go-api-boilerplate/pkg/log"
	"github.com/vardius/go-api-boilerplate/pkg/mysql"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	buildinfo.PrintVersionOrContinue()

	ctx := context.Background()

	logger := log.New(config.Env.App.Environment)
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
		config.Env.GRPC.Host,
		config.Env.GRPC.Port,
		grpc_utils.ConnectionConfig{
			ConnTime:    config.Env.GRPC.ConnTime,
			ConnTimeout: config.Env.GRPC.ConnTimeout,
		},
		logger,
	)
	defer grpcAuthConn.Close()

	grpPubsubClient := pubsub_proto.NewPubSubClient(grpcPubSubConn)
	grpPushPullClient := pushpull_proto.NewPushPullClient(grpcPushPullConn)
	eventBus := eventbus.New(grpPubsubClient, grpPushPullClient, logger)
	tokenRepository := repository.NewTokenRepository(eventStore, eventBus)
	clientRepository := repository.NewClientRepository(eventStore, eventBus)
	tokenPersistenceRepository := persistence.NewTokenRepository(mysqlConnection)
	clientPersistenceRepository := persistence.NewClientRepository(mysqlConnection)
	tokenStore := oauth2.NewTokenStore(tokenPersistenceRepository, commandBus)
	clientStore := oauth2.NewClientStore(clientPersistenceRepository)
	manager := oauth2.NewManager(tokenStore, clientStore, []byte(config.Env.App.Secret))
	oauth2Server := oauth2.InitServer(manager, mysqlConnection, logger, config.Env.App.Secret)
	grpcHealthServer := grpc_health.NewServer()
	grpcAuthServer := auth_grpc.NewServer(oauth2Server, logger, config.Env.App.Secret)
	router := auth_http.NewRouter(
		logger,
		oauth2Server,
		mysqlConnection,
		map[string]*grpc.ClientConn{
			"pushpull": grpcPushPullConn,
			"pubsub":   grpcPubSubConn,
			"auth":     grpcAuthConn,
		},
	)
	app := application.New(logger)

	// store our internal user service client
	if err := clientStore.SetInternal(config.Env.User.ClientID, &oauth2_models.Client{
		ID:     config.Env.User.ClientID,
		Secret: config.Env.User.ClientSecret,
		Domain: fmt.Sprintf("http://%s:%d", config.Env.User.Host, config.Env.HTTP.Port),
	}); err != nil {
		panic(err)
	}

	commandBus.Subscribe((token.Create{}).GetName(), token.OnCreate(tokenRepository, mysqlConnection))
	commandBus.Subscribe((token.Remove{}).GetName(), token.OnRemove(tokenRepository, mysqlConnection))
	commandBus.Subscribe((client.Create{}).GetName(), client.OnCreate(clientRepository, mysqlConnection))
	commandBus.Subscribe((client.Remove{}).GetName(), client.OnRemove(clientRepository, mysqlConnection))

	go func() {
		go func() {
			eventbus.RegisterHandlers(
				grpcPubSubConn,
				grpcPushPullConn,
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
