// Command auth is the auth service: sessions for the web app, OAuth 2 for
// third-party apps, and the keys that verify every access token.
//
//	auth              serve
//	auth migrate      apply the store's migrations and exit
//	auth env          print the configuration reference as Markdown
//	auth healthcheck  exit 0 if the service on this host is ready, 1 if not
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/vardius/go-api-boilerplate/gen/user/v1/userv1connect"
	"github.com/vardius/go-api-boilerplate/internal/auth/app"
	"github.com/vardius/go-api-boilerplate/internal/auth/httpapi"
	"github.com/vardius/go-api-boilerplate/internal/auth/keys"
	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/config"
	"github.com/vardius/go-api-boilerplate/internal/platform/connectx"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
	"github.com/vardius/go-api-boilerplate/internal/platform/obs"
	"github.com/vardius/go-api-boilerplate/internal/platform/run"
)

// version is set at build time with -ldflags "-X main.version=…".
var version = "dev"

func main() {
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	var err error
	switch cmd {
	case "serve":
		err = serve(context.Background())
	case "migrate":
		err = migrate(context.Background())
	case "env":
		fmt.Print(config.Markdown(config.Describe[Config]()))
	case "healthcheck":
		err = healthcheck(context.Background())
	default:
		err = fmt.Errorf("unknown command %q: want serve, migrate, env or healthcheck", cmd)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "auth:", err)
		os.Exit(1)
	}
}

func serve(ctx context.Context) error {
	cfg, err := config.Load[Config]()
	if err != nil {
		return err
	}
	log := obs.NewLogger(os.Stderr, cfg.Log)
	slog.SetDefault(log)
	shutdown, err := obs.Setup(ctx, "auth", version)
	if err != nil {
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			log.Error("flush telemetry", "err", err)
		}
	}()

	backend, err := openStore(ctx, cfg.Store, log)
	if err != nil {
		return err
	}
	defer backend.close()
	if cfg.MigrateOnStart {
		if err := backend.migrate(ctx); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	k, err := keys.Load(cfg.Keys, cfg.AppEnv == development, log)
	if err != nil {
		return err
	}
	signer := keys.NewSigner(k, cfg.Issuer, cfg.Audience)

	rpc := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport), Timeout: 10 * time.Second}
	tokens := keys.NewServiceToken(signer, time.Now)
	logins := userv1connect.NewLoginServiceClient(connect.NewClient(connecthttp.NewTransport(rpc, cfg.UserRPCURL), connectx.Bearer(tokens.Token)))

	probe := run.NewProbe(backend.checks)
	router, err := httpapi.New(httpapi.Config{
		App:               app.New(backend.store, signer, logins, time.Now),
		Keys:              k,
		Probe:             probe,
		Verifier:          signer.Verifier(),
		Issuer:            cfg.Issuer,
		WebURL:            cfg.WebURL,
		CookieSecure:      cfg.CookieSecure,
		RefreshCookiePath: cfg.RefreshCookiePath,
		TrustedProxies:    cfg.TrustedProxies,
		RateLimit:         cfg.RateLimit,
		Logger:            log,
	})
	if err != nil {
		return err
	}
	log.Info("serving", "version", version, "addr", cfg.HTTP.Addr, "issuer", cfg.Issuer)
	servers := []run.Server{run.HTTP(httpx.NewServer(cfg.HTTP, router))}
	if backend.purger != nil {
		servers = append(servers, store.PurgeWorker(backend.purger, purgeInterval, time.Now, log))
	}
	return run.Run(ctx, cfg.Run, probe, log, servers...)
}

func migrate(ctx context.Context) error {
	cfg, err := config.Load[migrateConfig]()
	if err != nil {
		return err
	}
	log := obs.NewLogger(os.Stderr, cfg.Log)
	backend, err := openStore(ctx, cfg.Store, log)
	if err != nil {
		return err
	}
	defer backend.close()
	if err := backend.migrate(ctx); err != nil {
		return err
	}
	log.Info("migrated", "version", version)
	return nil
}
