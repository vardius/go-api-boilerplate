package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/caarlos0/env"
	http_cors "github.com/rs/cors"
	auth_proto "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/proto"
	"github.com/vardius/go-api-boilerplate/cmd/user/application"
	"github.com/vardius/go-api-boilerplate/cmd/user/domain/user"
	user_persistence "github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/persistence/mysql"
	user_proto "github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/proto"
	user_repository "github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/repository"
	user_grpc "github.com/vardius/go-api-boilerplate/cmd/user/interfaces/grpc"
	user_http "github.com/vardius/go-api-boilerplate/cmd/user/interfaces/http"
	commandbus "github.com/vardius/go-api-boilerplate/pkg/commandbus/memory"
	eventbus "github.com/vardius/go-api-boilerplate/pkg/eventbus/memory"
	eventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore/memory"
	"github.com/vardius/go-api-boilerplate/pkg/grpc"
	http_recovery "github.com/vardius/go-api-boilerplate/pkg/http/recovery"
	http_response "github.com/vardius/go-api-boilerplate/pkg/http/response"
	http_authenticator "github.com/vardius/go-api-boilerplate/pkg/http/security/authenticator"
	"github.com/vardius/go-api-boilerplate/pkg/log"
	"github.com/vardius/go-api-boilerplate/pkg/mysql"
	os_shutdown "github.com/vardius/go-api-boilerplate/pkg/os/shutdown"
	gorouter "github.com/vardius/gorouter/v4"
	"golang.org/x/oauth2"
	grpc_health "google.golang.org/grpc/health"
	grpc_health_proto "google.golang.org/grpc/health/grpc_health_v1"
)

type config struct {
	Env          string   `env:"ENV"           envDefault:"development"`
	Secret       string   `env:"SECRET"        envDefault:"secret"`
	Origins      []string `env:"ORIGINS"       envSeparator:"|"` // Origins should follow format: scheme "://" host [ ":" port ]
	Host         string   `env:"HOST"          envDefault:"0.0.0.0"`
	ClientID     string   `env:"CLIENT_ID"     envDefault:"clientId"`
	ClientSecret string   `env:"CLIENT_SECRET" envDefault:"clientSecret"`
	PortHTTP     int      `env:"PORT_HTTP"     envDefault:"3020"`
	PortGRPC     int      `env:"PORT_GRPC"     envDefault:"3021"`
	DbHost       string   `env:"DB_HOST"       envDefault:"0.0.0.0"`
	DbPort       int      `env:"DB_PORT"       envDefault:"3306"`
	DbUser       string   `env:"DB_USER"       envDefault:"root"`
	DbPass       string   `env:"DB_PASS"       envDefault:"password"`
	DbName       string   `env:"DB_NAME"       envDefault:"goapiboilerplate"`
	AuthHost     string   `env:"AUTH_HOST"     envDefault:"0.0.0.0"`
}

func main() {
	ctx := context.Background()

	cfg := config{}
	env.Parse(&cfg)

	logger := log.New(cfg.Env)
	grpcServer := grpc.NewServer(logger)

	db := mysql.NewConnection(ctx, cfg.DbHost, cfg.DbPort, cfg.DbUser, cfg.DbPass, cfg.DbName, logger)
	defer db.Close()

	oauth2Config := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Scopes:       []string{"all"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("http://%s:%d/%s", cfg.AuthHost, cfg.PortHTTP, "authorize"),
			TokenURL: fmt.Sprintf("http://%s:%d/%s", cfg.AuthHost, cfg.PortHTTP, "token"),
		},
	}

	eventStore := eventstore.New()
	commandBus := commandbus.NewLoggable(runtime.NumCPU(), logger)
	eventBus := eventbus.NewLoggable(runtime.NumCPU(), logger)

	userRepository := user_repository.NewUserRepository(eventStore, eventBus)
	userMYSQLRepository := user_persistence.NewUserRepository(db)

	commandBus.Subscribe(fmt.Sprintf("%T", &user.RegisterWithEmail{}), user.OnRegisterWithEmail(userRepository, db))
	commandBus.Subscribe(fmt.Sprintf("%T", &user.RegisterWithGoogle{}), user.OnRegisterWithGoogle(userRepository, db))
	commandBus.Subscribe(fmt.Sprintf("%T", &user.RegisterWithFacebook{}), user.OnRegisterWithFacebook(userRepository, db))
	commandBus.Subscribe(fmt.Sprintf("%T", &user.ChangeEmailAddress{}), user.OnChangeEmailAddress(userRepository, db))
	commandBus.Subscribe(fmt.Sprintf("%T", &user.RequestAccessToken{}), user.OnRequestAccessToken(userRepository, db))

	eventBus.Subscribe(fmt.Sprintf("%T", &user.WasRegisteredWithEmail{}), application.WhenUserWasRegisteredWithEmail(db, userMYSQLRepository))
	eventBus.Subscribe(fmt.Sprintf("%T", &user.WasRegisteredWithGoogle{}), application.WhenUserWasRegisteredWithGoogle(db, userMYSQLRepository))
	eventBus.Subscribe(fmt.Sprintf("%T", &user.WasRegisteredWithFacebook{}), application.WhenUserWasRegisteredWithFacebook(db, userMYSQLRepository))
	eventBus.Subscribe(fmt.Sprintf("%T", &user.EmailAddressWasChanged{}), application.WhenUserEmailAddressWasChanged(db, userMYSQLRepository))
	eventBus.Subscribe(fmt.Sprintf("%T", &user.AccessTokenWasRequested{}), application.WhenUserAccessTokenWasRequested(oauth2Config, cfg.Secret))

	userServer := user_grpc.NewServer(commandBus, db)

	authConn := grpc.NewConnection(ctx, cfg.AuthHost, cfg.PortGRPC, logger)
	defer authConn.Close()

	userConn := grpc.NewConnection(ctx, cfg.Host, cfg.PortGRPC, logger)
	defer userConn.Close()

	grpAuthClient := auth_proto.NewAuthenticationServiceClient(authConn)
	grpUserClient := user_proto.NewUserServiceClient(userConn)

	healthServer := grpc_health.NewServer()
	healthServer.SetServingStatus("user", grpc_health_proto.HealthCheckResponse_SERVING)

	auth := http_authenticator.NewToken(application.TokenAuthHandler(grpAuthClient, user_persistence.NewUserRepository(db), logger))

	// Global middleware
	router := gorouter.New(
		logger.LogRequest,
		http_cors.Default().Handler,
		http_response.WithXSS,
		http_response.WithHSTS,
		http_response.AsJSON,
		auth.FromHeader("USER"),
		auth.FromQuery("authToken"),
		http_recovery.WithLogger(logger).RecoverHandler,
	)

	user_proto.RegisterUserServiceServer(grpcServer, userServer)
	grpc_health_proto.RegisterHealthServer(grpcServer, healthServer)

	user_http.AddHealthCheckRoutes(router, logger, userConn, authConn, db)
	user_http.AddAuthRoutes(router, grpUserClient, oauth2Config, cfg.Secret)
	user_http.AddUserRoutes(router, grpUserClient)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.PortHTTP),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      router,
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.PortGRPC))
	if err != nil {
		logger.Critical(ctx, "tcp failed to listen %s:%d\n%v\n", cfg.Host, cfg.PortGRPC, err)
		os.Exit(1)
	}

	stop := func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		logger.Info(ctx, "shutting down...\n")

		grpcServer.GracefulStop()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Critical(ctx, "shutdown error: %v\n", err)
		} else {
			logger.Info(ctx, "gracefully stopped\n")
		}
	}

	go func() {
		logger.Critical(ctx, "failed to serve: %v\n", grpcServer.Serve(lis))
		stop()
		os.Exit(1)
	}()

	go func() {
		logger.Critical(ctx, "%v\n", srv.ListenAndServe())
		stop()
		os.Exit(1)
	}()

	logger.Info(ctx, "tcp running at %s:%d\n", cfg.Host, cfg.PortGRPC)
	logger.Info(ctx, "http running at %s:%d\n", cfg.Host, cfg.PortHTTP)

	os_shutdown.GracefulStop(stop)
}
