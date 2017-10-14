package main

import (
	"net/http"
	"app/pkg/auth"
	"app/pkg/controller"
	"app/pkg/domain/user"
	"app/pkg/dynamodb"
	"app/pkg/memory"
	"app/pkg/middleware"
	"strconv"
	"time"

	"github.com/caarlos0/env"
	"github.com/vardius/golog"
	"github.com/vardius/gorouter"
)

type config struct {
	Env      string   `env:"API_ENV" envDefault:"development"`
	Host     string   `env:"API_HOST" envDefault:"localhost"`
	Port     int      `env:"PORT" envDefault:"3000"`
	Realm    string   `env:"API_REALM" envDefault:"API"`
	CertPath string   `env:"API_CERT_PATH"`
	KeyPath  string   `env:"API_KEY_PATH"`
	Secret   string   `env:"API_SECRET" envDefault:"secret"`
	Origins  []string `env:"API_ORIGINS" envSeparator:"|"` // Origins should follow format: scheme "://" host [ ":" port ]
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

	logger := golog.New(getLogLevelByEnv(cfg.Env))
	eventStore := memory.NewEventStore()
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
	router.POST("/auth/google/callback", controller.GoogleAuth(commandBus, jwtService))
	router.POST("/auth/facebook/callback", controller.FacebookAuth(commandBus, jwtService))

	// Applies middleware to self and all children routes
	// router.USE(gorouter.POST, "/dispatch", middleware.Bearer(cfg.Realm, jwtService.Authenticate))

	if cfg.CertPath != "" && cfg.KeyPath != "" {
		logger.Critical(nil, "%v\n", http.ListenAndServeTLS(":"+strconv.Itoa(cfg.Port), cfg.CertPath, cfg.KeyPath, router))
	} else {
		logger.Critical(nil, "%v\n", http.ListenAndServe(":"+strconv.Itoa(cfg.Port), router))
	}
}
