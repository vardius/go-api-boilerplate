package main

import (
	"crypto/tls"
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
	"golang.org/x/crypto/acme/autocert"
)

type config struct {
	Env          string   `env:"ENV"          envDefault:"development"`
	Host         string   `env:"HOST"         envDefault:"localhost"`
	Port         int      `env:"PORT"         envDefault:"3000"`
	CertDirCache string   `env:"CERT_DIR_CACHE"`
	Secret       string   `env:"SECRET"       envDefault:"secret"`
	Origins      []string `env:"ORIGINS"      envSeparator:"|"` // Origins should follow format: scheme "://" host [ ":" port ]
	AwsRegion    string   `env:"AWS_REGION"   envDefault:"us-east-1"`
	AwsEndpoint  string   `env:"AWS_ENDPOINT" envDefault:"http://localhost:4569"`
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

	// Global middleware
	router := gorouter.New(
		logger.LogRequest,
		cors.Default().Handler,
		nosurf.NewPure,
		response.XSS,
		response.HSTS,
		response.JSON,
		auth.Bearer("API", jwtService.Decode),
		auth.Query("authToken", jwtService.Decode),
		recover.WithLogger(recover.New(), logger).WrapHandler,
	)

	userDomain := user.NewDomain(
		commandBus,
		eventBus,
		eventStore,
		jwtService,
	)
	// User domain
	router.Mount("/users", userDomain.AsRouter())

	// Routes
	// Social media auth routes
	router.POST("/auth/google/callback", socialmedia.NewGoogle(commandBus, jwtService))
	router.POST("/auth/facebook/callback", socialmedia.NewFacebook(commandBus, jwtService))

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(cfg.Host),
		Cache:      autocert.DirCache(cfg.CertDirCache),
	}

	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		GetCertificate: certManager.GetCertificate,
	}

	srv := &http.Server{
		Addr:         ":" + strconv.Itoa(cfg.Port),
		Handler:      router,
		TLSConfig:    tlsConfig,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	logger.Critical(nil, "%v\n", srv.ListenAndServeTLS("", ""))
}
