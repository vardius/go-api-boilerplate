package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/caarlos0/env"
	"github.com/justinas/nosurf"
	"github.com/rs/cors"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	"github.com/vardius/go-api-boilerplate/pkg/auth/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/auth/socialmedia"
	"github.com/vardius/go-api-boilerplate/pkg/aws/dynamodb/eventstore"
	"github.com/vardius/go-api-boilerplate/pkg/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/http/recover"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/log"
	"github.com/vardius/go-api-boilerplate/pkg/memory/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/memory/eventbus"
	"github.com/vardius/gorouter"
)

type config struct {
	Env         string   `env:"ENV"          envDefault:"development"`
	Host        string   `env:"HOST"         envDefault:"localhost"`
	Port        int      `env:"PORT"         envDefault:"3000"`
	Realm       string   `env:"REALM"        envDefault:"API"`
	CertPath    string   `env:"CERT_PATH"`
	KeyPath     string   `env:"KEY_PATH"`
	Secret      string   `env:"SECRET"       envDefault:"secret"`
	Origins     []string `env:"ORIGINS"      envSeparator:"|"` // Origins should follow format: scheme "://" host [ ":" port ]
	AwsRegion   string   `env:"AWS_REGION"   envDefault:"us-east-1"`
	AwsEndpoint string   `env:"AWS_ENDPOINT" envDefault:"http://localhost:4569"`
}

func main() {
	cfg := config{}
	env.Parse(&cfg)

	awsConfig := &aws.Config{
		Region:   aws.String(cfg.AwsRegion),
		Endpoint: aws.String(cfg.AwsEndpoint),
	}

	logger := log.New(cfg.Env)
	jwtService := jwt.New([]byte(cfg.Secret), time.Hour*24)
	eventStore := eventstore.New("events", awsConfig)
	eventBus := eventbus.WithLogger(eventbus.New(), logger)
	commandBus := commandbus.WithLogger(commandbus.New(), logger)

	// Domains
	user.Init(eventStore, eventBus, commandBus, jwtService)

	// Global middleware
	router := gorouter.New(
		logger.LogRequest,
		cors.Default().Handler,
		nosurf.NewPure,
		response.XSS,
		response.JSON,
		recover.WithLogger(recover.New(), logger).WrapHandler,
	)

	// Routes
	// Social media auth routes
	router.POST("/auth/google/callback", socialmedia.NewGoogle(commandBus, jwtService))
	router.POST("/auth/facebook/callback", socialmedia.NewFacebook(commandBus, jwtService))

	// User domain
	router.POST("/dispatch/users/{command}", user.NewDispatcher(commandBus))
	// User domain routes middleware
	// Applies middleware to itself and all children routes
	router.USE(gorouter.POST, "/dispatch/users/"+user.ChangeEmailAddress, auth.Bearer(cfg.Realm, jwtService.Decode))

	if cfg.CertPath != "" && cfg.KeyPath != "" {
		logger.Critical(nil, "%v\n", http.ListenAndServeTLS(":"+strconv.Itoa(cfg.Port), cfg.CertPath, cfg.KeyPath, router))
	} else {
		logger.Critical(nil, "%v\n", http.ListenAndServe(":"+strconv.Itoa(cfg.Port), router))
	}
}
