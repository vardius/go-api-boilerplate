package main

import (
	"context"
	"net/netip"
	"net/url"

	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/es"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
	"github.com/vardius/go-api-boilerplate/internal/platform/obs"
	"github.com/vardius/go-api-boilerplate/internal/platform/outbox"
	"github.com/vardius/go-api-boilerplate/internal/platform/run"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
	"github.com/vardius/go-api-boilerplate/internal/user/httpapi"
	"github.com/vardius/go-api-boilerplate/internal/user/mail"
	"github.com/vardius/go-api-boilerplate/internal/user/store"
)

// Config is the service's configuration, read from the environment.
type Config struct {
	HTTP           httpx.ServerConfig `envPrefix:"HTTP_"`
	RPCAddr        string             `env:"RPC_ADDR" envDefault:":8081" desc:"Address the Connect API listens on, for the auth service only: keep it off the gateway."`
	Log            obs.Config         `envPrefix:"LOG_"`
	Run            run.Config
	WebURL         url.URL               `env:"WEB_URL,required" desc:"The web app's public origin, which mailed links and redirects lead to."`
	Auth           authn.VerifierConfig  `envPrefix:"AUTH_"`
	SMTP           mail.SMTPConfig       `envPrefix:"SMTP_"`
	Google         httpapi.GoogleConfig  `envPrefix:"GOOGLE_"`
	MailPreviewURL string                `env:"MAIL_PREVIEW_URL" desc:"A mail catcher's web UI for the sign-in page to link to, such as Mailpit in development. Empty in production."`
	TrustedProxies []netip.Prefix        `env:"TRUSTED_PROXIES" desc:"Comma-separated CIDRs of the proxies whose X-Forwarded-For names the client, as the login-link rate limit needs."`
	RateLimit      httpx.RateLimitConfig `envPrefix:"RATE_LIMIT_"`
	Outbox         outbox.RelayConfig    `envPrefix:"OUTBOX_"`
	MigrateOnStart bool                  `env:"MIGRATE_ON_START" envDefault:"false" desc:"Apply the migrations (MySQL) or ensure the indexes (MongoDB) before serving."`
	Store          storeConfig
}

// stores is what openStores opens for the backend this binary is built with.
type stores struct {
	uow     uow.UnitOfWork
	events  es.Store
	outbox  outbox.Store
	users   store.Users
	codes   store.LoginCodes
	checks  map[string]run.Check
	migrate func(ctx context.Context) error
	close   func() error
}
