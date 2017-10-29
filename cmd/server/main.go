package main

import (
	"app/pkg/auth"
	"app/pkg/auth/jwt"
	"app/pkg/auth/socialmedia"
	"app/pkg/aws/dynamodb"
	"app/pkg/domain"
	"app/pkg/domain/user"
	"app/pkg/http/recover"
	"app/pkg/http/response"
	"app/pkg/log"
	"app/pkg/memory"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/caarlos0/env"
	"github.com/justinas/nosurf"
	"github.com/rs/cors"
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
	eventStore := dynamodb.NewEventStore("events", awsConfig)
	eventBus := memory.NewEventBus(logger)
	commandBus := memory.NewCommandBus(logger)

	// Domains
	user.Init(eventStore, eventBus, commandBus, jwtService)

	// Global middleware
	router := gorouter.New(
		logger.LogRequest,
		cors.Default().Handler,
		nosurf.NewPure,
		response.XSS,
		response.JSON,
		recover.New(logger),
	)

	// Routes
	// Social media auth routes
	router.POST("/auth/google/callback", socialmedia.NewGoogle(commandBus, jwtService))
	router.POST("/auth/facebook/callback", socialmedia.NewFacebook(commandBus, jwtService))
	// Command dispatch route
	router.POST("/dispatch/{domain}/{command}", domain.NewDispatcher(commandBus))

	// Routes middleware
	// Applies middleware to itself and all children routes
	router.USE(gorouter.POST, "/dispatch/"+user.Domain+"/"+user.ChangeEmailAddress, auth.Bearer(cfg.Realm, jwtService.Decode))

	if cfg.CertPath != "" && cfg.KeyPath != "" {
		logger.Critical(nil, "%v\n", http.ListenAndServeTLS(":"+strconv.Itoa(cfg.Port), cfg.CertPath, cfg.KeyPath, router))
	} else {
		logger.Critical(nil, "%v\n", http.ListenAndServe(":"+strconv.Itoa(cfg.Port), router))
	}
}
