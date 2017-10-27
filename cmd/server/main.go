package main

import (
	"app/pkg/auth"
	"app/pkg/domain"
	"app/pkg/domain/user"
	"app/pkg/dynamodb"
	"app/pkg/err"
	"app/pkg/json"
	"app/pkg/jwt"
	"app/pkg/log"
	"app/pkg/memory"
	"app/pkg/nosniff"
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
	j := jwt.New([]byte(cfg.Secret), time.Hour*24)
	eventStore := dynamodb.NewEventStore("events", awsConfig)
	eventBus := memory.NewEventBus(logger)
	commandBus := memory.NewCommandBus(logger)

	user.Init(eventStore, eventBus, commandBus, j)

	router := gorouter.New(
		logger.LogRequest,
		cors.Default().Handler,
		nosurf.NewPure,
		nosniff.XSSHeader,
		json.Parse,
		err.NewPanicRecover(logger),
	)

	router.POST("/dispatch/{domain}/{command}", domain.NewDispatcher(commandBus))
	router.POST("/auth/google/callback", auth.NewGoogleAuth(commandBus, j))
	router.POST("/auth/facebook/callback", auth.NewFacebookAuth(commandBus, j))

	// Applies middleware to self and all children routes
	// router.USE(gorouter.POST, "/dispatch", auth.Bearer(cfg.Realm, j.Authenticate))

	if cfg.CertPath != "" && cfg.KeyPath != "" {
		logger.Critical(nil, "%v\n", http.ListenAndServeTLS(":"+strconv.Itoa(cfg.Port), cfg.CertPath, cfg.KeyPath, router))
	} else {
		logger.Critical(nil, "%v\n", http.ListenAndServe(":"+strconv.Itoa(cfg.Port), router))
	}
}
