package main

import (
	"context"
	"crypto/tls"
	"net/http"
	"strconv"
	"time"

	"github.com/caarlos0/env"
	"github.com/rs/cors"
	"github.com/vardius/go-api-boilerplate/pkg/common/calm"
	"github.com/vardius/go-api-boilerplate/pkg/common/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/common/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/common/log"
	"github.com/vardius/go-api-boilerplate/pkg/common/os/shutdown"
	"github.com/vardius/go-api-boilerplate/pkg/common/security/authenticator"
	user_grpc_client "github.com/vardius/go-api-boilerplate/pkg/proxy/infrastructure/user/grpc"
	proxy_http_server "github.com/vardius/go-api-boilerplate/pkg/proxy/interfaces/http"
	"github.com/vardius/gorouter"
	"golang.org/x/crypto/acme/autocert"
)

type config struct {
	Env          string   `env:"ENV"             envDefault:"development"`
	Host         string   `env:"HOST"            envDefault:"localhost"`
	Port         int      `env:"PORT"            envDefault:"3000"`
	UserHost     string   `env:"USER_HOST"       envDefault:"localhost"`
	UserPort     int      `env:"USER_PORT"       envDefault:"3001"`
	CertDirCache string   `env:"CERT_DIR_CACHE"`
	Secret       string   `env:"SECRET"          envDefault:"secret"`
	Origins      []string `env:"ORIGINS"         envSeparator:"|"` // Origins should follow format: scheme "://" host [ ":" port ]
}

func main() {
	ctx := context.Background()

	cfg := config{}
	env.Parse(&cfg)

	logger := log.New(cfg.Env)
	clm := calm.WithLogger(calm.New(), logger)
	jwtService := jwt.New([]byte(cfg.Secret), time.Hour*24)
	auth := authenticator.WithToken(jwtService.Decode)

	grpUserClient := user_grpc_client.New(cfg.UserHost, cfg.UserPort)

	// Global middleware
	router := gorouter.New(
		logger.LogRequest("proxy"),
		cors.Default().Handler,
		response.WithXSS,
		response.WithHSTS,
		response.AsJSON,
		auth.FromHeader("API"),
		auth.FromQuery("authToken"),
		clm.RecoverHandler,
	)

	proxy_http_server.AddUserRoutes(router, grpUserClient, jwtService)

	srv := setupServer(&cfg, router)

	go func() {
		logger.Critical(ctx, "%v\n", srv.ListenAndServeTLS("", ""))
	}()

	logger.Info(ctx, "[proxy] running at %s:%d\n", cfg.Host, cfg.Port)

	shutdown.GracefulStop(func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		logger.Info(ctx, "[proxy] shutting down...\n")

		if err := srv.Shutdown(ctx); err != nil {
			logger.Info(ctx, "[proxy] shutdown error: %v\n", err)
		} else {
			logger.Info(ctx, "[proxy] gracefully stopped\n")
		}
	})
}

func setupServer(cfg *config, router gorouter.Router) *http.Server {
	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.Port),
		Handler: router,
	}

	// for localhost do not use autocert
	// https://github.com/vardius/go-api-boilerplate/issues/2
	if cfg.Host == "localhost" {
		return srv
	}

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

	srv.TLSConfig = tlsConfig
	srv.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0)

	return srv
}
