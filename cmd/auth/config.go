package main

import (
	"net/netip"

	"github.com/vardius/go-api-boilerplate/internal/auth/keys"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
	"github.com/vardius/go-api-boilerplate/internal/platform/obs"
	"github.com/vardius/go-api-boilerplate/internal/platform/run"
)

// development is the APP_ENV that may run on an ephemeral signing key.
const development = "development"

// Config is what the service reads from the environment. `auth env`
// prints it as the configuration reference.
type Config struct {
	AppEnv string             `env:"APP_ENV" envDefault:"production" desc:"Where the service runs. development lets it start without SIGNING_KEY_FILE, on a key generated at start that dies with the process."`
	HTTP   httpx.ServerConfig `envPrefix:"HTTP_"`
	Log    obs.Config         `envPrefix:"LOG_"`
	Run    run.Config

	Issuer   string `env:"ISSUER,required" desc:"Public URL of the auth API, the iss of every token, such as http://localhost:8080/api/auth."`
	Audience string `env:"AUDIENCE" envDefault:"go-api-boilerplate" desc:"The aud of every access token; every service checks it."`
	WebURL   string `env:"WEB_URL,required" desc:"Public URL of the web app, such as http://localhost:8080, which hosts the sign-in, consent and OAuth error pages."`
	Keys     keys.Config

	CookieSecure      bool   `env:"COOKIE_SECURE" envDefault:"true" desc:"Mark the session cookies Secure. Turn off only to serve plain http on localhost."`
	RefreshCookiePath string `env:"REFRESH_COOKIE_PATH" envDefault:"/api/auth/v1/sessions" desc:"Public path of the sessions routes: the only path the refresh cookie is sent to."`

	UserRPCURL     string                `env:"USER_RPC_URL,required" desc:"Base URL of the user service's Connect API, such as http://user:8081, where sign-in codes are redeemed."`
	TrustedProxies []netip.Prefix        `env:"TRUSTED_PROXIES" desc:"Comma-separated CIDRs of the proxies whose X-Forwarded-For names the client, for rate limits."`
	RateLimit      httpx.RateLimitConfig `envPrefix:"RATE_LIMIT_"`

	Store          storeConfig
	MigrateOnStart bool `env:"MIGRATE_ON_START" envDefault:"false" desc:"Apply the store's migrations before serving. Leave off where a separate job migrates, as the Helm chart does."`
}

// migrateConfig is what `auth migrate` reads: the Helm migration job sets
// nothing else.
type migrateConfig struct {
	Log   obs.Config `envPrefix:"LOG_"`
	Store storeConfig
}

// healthcheckConfig is what `auth healthcheck` reads, to find the server.
type healthcheckConfig struct {
	HTTP httpx.ServerConfig `envPrefix:"HTTP_"`
}
