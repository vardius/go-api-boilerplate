package main

import (
	"app/pkg/auth"
	"app/pkg/controller"
	"app/pkg/domain/user"
	"app/pkg/dynamodb"
	"app/pkg/memory"
	"app/pkg/middleware"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/caarlos0/env"
	"github.com/vardius/golog"
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

func getLogLevelByEnv(env string) string {
	logLevel := "info"
	if env == "development" {
		logLevel = "debug"
	}

	return logLevel
}

func main() {
	cfg := config{}
	env.Parse(&cfg)

	awsConfig := &aws.Config{
		Region:   aws.String(cfg.AwsRegion),
		Endpoint: aws.String(cfg.AwsEndpoint),
	}

	logger := golog.New(getLogLevelByEnv(cfg.Env))
	// logger := golog.NewFileLogger(getLogLevelByEnv(cfg.Env), "/tmp/prod.log")
	eventStore := dynamodb.NewEventStore("events", awsConfig)
	eventBus := memory.NewEventBus(logger)
	commandBus := memory.NewCommandBus(logger)
	jwtService := auth.NewJwtService([]byte(cfg.Secret), time.Hour*24)

	user.Init(eventStore, eventBus, commandBus, jwtService, logger)

	router := gorouter.New(
		middleware.NewLogger(logger),
		// middleware.NewCors(cfg.Origins), //todo: uncomment
		middleware.XSSHeader,
		middleware.JSONHeader,
		middleware.JSONBody,
		middleware.NewPanicRecover(logger),
	)

	router.POST("/dispatch/{domain}/{command}", controller.CommandDispatch(commandBus))
	router.POST("/auth/google/callback", controller.NewGoogleAuth(commandBus, jwtService))
	router.POST("/auth/facebook/callback", controller.NewFacebookAuth(commandBus, jwtService))

	// Applies middleware to self and all children routes
	// router.USE(gorouter.POST, "/dispatch", middleware.Bearer(cfg.Realm, jwtService.Authenticate))

	if cfg.CertPath != "" && cfg.KeyPath != "" {
		logger.Critical(nil, "%v\n", http.ListenAndServeTLS(":"+strconv.Itoa(cfg.Port), cfg.CertPath, cfg.KeyPath, router))
	} else {
		logger.Critical(nil, "%v\n", http.ListenAndServe(":"+strconv.Itoa(cfg.Port), router))
	}
}
