// Command user is the user service: sign-in links, the signed-in user,
// email change and Google sign-in over HTTP, and login-code redemption over
// Connect for the auth service.
//
//	user              serve (the default)
//	user migrate      apply the store's migrations or indexes, then exit
//	user env          print the configuration reference as Markdown
//	user healthcheck  exit 0 when the service on this host is ready
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/config"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
	"github.com/vardius/go-api-boilerplate/internal/platform/obs"
	"github.com/vardius/go-api-boilerplate/internal/platform/outbox"
	"github.com/vardius/go-api-boilerplate/internal/platform/run"
	"github.com/vardius/go-api-boilerplate/internal/user/app"
	"github.com/vardius/go-api-boilerplate/internal/user/httpapi"
	"github.com/vardius/go-api-boilerplate/internal/user/mail"
	"github.com/vardius/go-api-boilerplate/internal/user/rpc"
)

// version is set at build time: -ldflags "-X main.version=…".
var version = "dev"

// outboundTimeout bounds every call to the auth service's JWKS and to
// Google.
const outboundTimeout = 10 * time.Second

func main() {
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	var err error
	switch ctx := context.Background(); cmd {
	case "serve":
		err = serve(ctx)
	case "migrate":
		err = migrate(ctx)
	case "env":
		fmt.Print(config.Markdown(config.Describe[Config]()))
	case "healthcheck":
		err = healthcheck(ctx)
	default:
		err = fmt.Errorf("unknown command %q: want serve, migrate, env or healthcheck", cmd)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "user:", err)
		os.Exit(1)
	}
}

func serve(ctx context.Context) (err error) {
	cfg, err := config.Load[Config]()
	if err != nil {
		return err
	}
	if !cfg.WebURL.IsAbs() || cfg.WebURL.Host == "" {
		return fmt.Errorf("WEB_URL %q is not an absolute URL", cfg.WebURL.String())
	}
	log := obs.NewLogger(os.Stdout, cfg.Log)
	slog.SetDefault(log)
	shutdown, err := obs.Setup(ctx, "user", version)
	if err != nil {
		return err
	}
	st, err := openStores(ctx, cfg.Store)
	if err != nil {
		return errors.Join(err, shutdown(ctx))
	}
	defer func() { err = errors.Join(err, st.close(), shutdown(context.WithoutCancel(ctx))) }()
	if cfg.MigrateOnStart {
		if err := st.migrate(ctx); err != nil {
			return err
		}
	}
	mailer, err := mail.NewSMTP(cfg.SMTP)
	if err != nil {
		return err
	}
	var google *httpapi.Google
	if cfg.Google.Enabled() {
		if google, err = httpapi.NewGoogle(cfg.Google, httpapi.GoogleIssuer, &http.Client{Timeout: outboundTimeout}); err != nil {
			return err
		}
	}

	svc := app.New(app.Deps{UnitOfWork: st.uow, Events: st.events, Outbox: st.outbox, Users: st.users, LoginCodes: st.codes, WebURL: &cfg.WebURL})
	verifier := authn.NewVerifier(cfg.Auth, &http.Client{Timeout: outboundTimeout})
	probe := run.NewProbe(st.checks)
	api := httpapi.New(httpapi.Config{
		App: svc, Probe: probe, Verifier: verifier, Google: google, MailPreviewURL: cfg.MailPreviewURL,
		TrustedProxies: cfg.TrustedProxies, RateLimit: cfg.RateLimit, Logger: log,
	})
	rpcServer := cfg.HTTP
	rpcServer.Addr = cfg.RPCAddr
	relay := &outbox.Relay{Store: st.outbox, Handlers: mail.Handlers(mailer, time.Now), Config: cfg.Outbox, Logger: log}

	log.Info("starting", "version", version, "http", cfg.HTTP.Addr, "rpc", cfg.RPCAddr, "google", google != nil)
	return run.Run(ctx, cfg.Run, probe, log,
		run.HTTP(httpx.NewServer(cfg.HTTP, api)),
		run.HTTP(httpx.NewServer(rpcServer, rpc.NewHandler(svc, verifier, log))),
		run.Worker(relay.Run),
		run.Worker(func(ctx context.Context) error { return purgeCodes(ctx, svc, log) }),
	)
}

func migrate(ctx context.Context) (err error) {
	cfg, err := config.Load[storeConfig]()
	if err != nil {
		return err
	}
	st, err := openStores(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, st.close()) }()
	return st.migrate(ctx)
}

// purgeEvery is how often expired login codes are deleted.
const purgeEvery = time.Hour

func purgeCodes(ctx context.Context, svc *app.Service, log *slog.Logger) error {
	tick := time.NewTicker(purgeEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			if err := svc.PurgeExpiredCodes(ctx); err != nil && ctx.Err() == nil {
				log.ErrorContext(ctx, "purge expired login codes", "err", err)
			}
		}
	}
}
