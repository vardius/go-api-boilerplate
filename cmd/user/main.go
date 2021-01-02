package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	"github.com/vardius/gocontainer"
	"google.golang.org/grpc"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/eventhandler"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/services"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	usergrpc "github.com/vardius/go-api-boilerplate/cmd/user/internal/interfaces/grpc"
	userhttp "github.com/vardius/go-api-boilerplate/cmd/user/internal/interfaces/http"
	userproto "github.com/vardius/go-api-boilerplate/cmd/user/proto"
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
		// @TODO: Secure grpc server with firewall
		nil, // []grpc.UnaryServerInterceptor{
		// 	firewall.GrantAccessForUnaryRequest(identity.RoleUser),
		// },
		nil, // []grpc.StreamServerInterceptor{
		// 	firewall.GrantAccessForStreamRequest(identity.RoleUser),
		// },
	)

	router := userhttp.NewRouter(
		cfg,
		container.Logger,
		container.TokenAuthorizer,
		container.UserPersistenceRepository,
		container.CommandBus,
		container.SQL,
		map[string]*grpc.ClientConn{
			"user": container.UserConn,
		},
	)

	if err := container.CommandBus.Subscribe(ctx, (user.RegisterWithEmail{}).GetName(), user.OnRegisterWithEmail(container.UserRepository, container.UserPersistenceRepository)); err != nil {
		panic(err)
	}
	if err := container.CommandBus.Subscribe(ctx, (user.RequestAccessToken{}).GetName(), user.OnRequestAccessToken(container.UserRepository)); err != nil {
		panic(err)
	}
	if err := container.CommandBus.Subscribe(ctx, (user.RegisterWithGoogle{}).GetName(), user.OnRegisterWithGoogle(container.UserRepository, container.UserPersistenceRepository)); err != nil {
		panic(err)
	}
	if err := container.CommandBus.Subscribe(ctx, (user.RegisterWithFacebook{}).GetName(), user.OnRegisterWithFacebook(container.UserRepository, container.UserPersistenceRepository)); err != nil {
		panic(err)
	}
	if err := container.CommandBus.Subscribe(ctx, (user.ChangeEmailAddress{}).GetName(), user.OnChangeEmailAddress(container.UserRepository, container.UserPersistenceRepository)); err != nil {
		panic(err)
	}

	if err := container.EventBus.Subscribe(ctx, (user.WasRegisteredWithEmail{}).GetType(), eventhandler.WhenUserWasRegisteredWithEmail(container.SQL, container.UserPersistenceRepository, container.CommandBus)); err != nil {
		panic(err)
	}
	if err := container.EventBus.Subscribe(ctx, (user.WasRegisteredWithGoogle{}).GetType(), eventhandler.WhenUserWasRegisteredWithGoogle(container.SQL, container.UserPersistenceRepository, container.CommandBus)); err != nil {
		panic(err)
	}
	if err := container.EventBus.Subscribe(ctx, (user.WasRegisteredWithFacebook{}).GetType(), eventhandler.WhenUserWasRegisteredWithFacebook(container.SQL, container.UserPersistenceRepository, container.CommandBus)); err != nil {
		panic(err)
	}
	if err := container.EventBus.Subscribe(ctx, (user.EmailAddressWasChanged{}).GetType(), eventhandler.WhenUserEmailAddressWasChanged(container.SQL, container.UserPersistenceRepository)); err != nil {
		panic(err)
	}
	if err := container.EventBus.Subscribe(ctx, (user.AccessTokenWasRequested{}).GetType(), eventhandler.WhenUserAccessTokenWasRequested(cfg, jwt.SigningMethodHS512, container.Authenticator, container.UserPersistenceRepository, container.AuthClient)); err != nil {
		panic(err)
	}
	if err := container.EventBus.Subscribe(ctx, (user.ConnectedWithGoogle{}).GetType(), eventhandler.WhenUserConnectedWithGoogle(container.SQL, container.UserPersistenceRepository, container.CommandBus)); err != nil {
		panic(err)
	}
	if err := container.EventBus.Subscribe(ctx, (user.ConnectedWithFacebook{}).GetType(), eventhandler.WhenUserConnectedWithFacebook(container.SQL, container.UserPersistenceRepository, container.CommandBus)); err != nil {
		panic(err)
	}

	grpcUserServer := usergrpc.NewServer(container.CommandBus, container.UserPersistenceRepository)
	userproto.RegisterUserServiceServer(grpcServer, grpcUserServer)

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
			"user",
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
