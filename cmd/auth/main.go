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
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/services"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/services/eventhandler"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/services/oauth2"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/client"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	authgrpc "github.com/vardius/go-api-boilerplate/cmd/auth/internal/interfaces/grpc"
	authhttp "github.com/vardius/go-api-boilerplate/cmd/auth/internal/interfaces/http"
	authproto "github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/buildinfo"
	grpcutils "github.com/vardius/go-api-boilerplate/pkg/grpc"
	httputils "github.com/vardius/go-api-boilerplate/pkg/http"
)

func init() {
	rand.Seed(time.Now().UnixNano())

	gocontainer.GlobalContainer = nil // disable global container instance
}

func main() {
	buildinfo.PrintVersionOrContinue()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.FromEnv()
	fmt.Println("CONFIG:", cfg)

	container, err := services.NewServiceContainer(ctx, cfg)
	if err != nil {
		panic(fmt.Errorf("failed to create service container: %w", err))
	}
	defer container.Close()

	grpcServer := grpcutils.NewServer(
		grpcutils.ServerConfig{
			ServerMinTime: cfg.GRPC.ServerMinTime,
			ServerTime:    cfg.GRPC.ServerTime,
			ServerTimeout: cfg.GRPC.ServerTimeout,
		},
		container.Logger,
		nil,
		nil,
	)

	oauth2Server := oauth2.InitServer(cfg, container.OAuth2Manager, container.Logger, container.UserPersistenceRepository, container.ClientPersistenceRepository, cfg.App.Secret, cfg.OAuth.InitTimeout)
	grpcAuthServer := authgrpc.NewServer(oauth2Server, container.ClientRepository, container.Authenticator)

	router := authhttp.NewRouter(
		cfg,
		container.Logger,
		container.TokenAuthorizer,
		oauth2Server,
		container.CommandBus,
		container.SQL,
		map[string]*grpc.ClientConn{
			"auth": container.AuthConn,
		},
		container.TokenPersistenceRepository,
		container.ClientPersistenceRepository,
	)

	// if err := container.CommandBus.Subscribe(ctx, (token.Create{}).GetName(), token.OnCreate(container.TokenRepository)); err != nil {
	// 	panic(err)
	// }
	if err := container.CommandBus.Subscribe(ctx, (token.Remove{}).GetName(), token.OnRemove(container.TokenRepository)); err != nil {
		panic(err)
	}
	if err := container.CommandBus.Subscribe(ctx, (client.Create{}).GetName(), client.OnCreate(container.ClientRepository)); err != nil {
		panic(err)
	}
	if err := container.CommandBus.Subscribe(ctx, (client.Remove{}).GetName(), client.OnRemove(container.ClientRepository)); err != nil {
		panic(err)
	}

	if err := container.EventBus.Subscribe(ctx, (token.WasCreated{}).GetType(), eventhandler.WhenTokenWasCreated(container.SQL, container.TokenPersistenceRepository)); err != nil {
		panic(err)
	}
	if err := container.EventBus.Subscribe(ctx, (token.WasRemoved{}).GetType(), eventhandler.WhenTokenWasRemoved(container.SQL, container.TokenPersistenceRepository)); err != nil {
		panic(err)
	}
	if err := container.EventBus.Subscribe(ctx, (client.WasCreated{}).GetType(), eventhandler.WhenClientWasCreated(container.SQL, container.ClientPersistenceRepository)); err != nil {
		panic(err)
	}
	if err := container.EventBus.Subscribe(ctx, (client.WasRemoved{}).GetType(), eventhandler.WhenClientWasRemoved(container.SQL, container.ClientPersistenceRepository)); err != nil {
		panic(err)
	}

	authproto.RegisterAuthenticationServiceServer(grpcServer, grpcAuthServer)

	app := application.New(container.Logger)
	app.AddAdapters(
		httputils.NewAdapter(
			&http.Server{
				Addr:         fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port),
				ReadTimeout:  cfg.HTTP.ReadTimeout,
				WriteTimeout: cfg.HTTP.WriteTimeout,
				IdleTimeout:  cfg.HTTP.IdleTimeout, // limits server-side the amount of time a Keep-Alive connection will be kept idle before being reused
				Handler:      router,
			},
		),
		grpcutils.NewAdapter(
			"auth",
			fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port),
			grpcServer,
		),
	)

	if cfg.App.Environment == "development" {
		app.AddAdapters(
			application.NewDebugAdapter(
				fmt.Sprintf("%s:%d", cfg.Debug.Host, cfg.Debug.Port),
			),
		)
	}

	app.WithShutdownTimeout(cfg.App.ShutdownTimeout)
	app.Run(ctx)
}
