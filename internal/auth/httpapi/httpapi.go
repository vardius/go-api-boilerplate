// Package httpapi is the auth service's HTTP API, the routes the gateway
// serves under /api/auth: sessions for the web app, OAuth client
// registration, the OAuth 2 authorization server, and the keys and
// metadata that let everyone else check its tokens. api/auth.openapi.yaml
// documents every route, and tests hold the two, and the authorization
// matrix, to the router's Routes.
//
// Session and consent routes are authenticated by cookie, so their unsafe
// methods sit behind httpx.CrossOrigin. The token and revocation endpoints
// authenticate the client instead and answer in RFC 6749's error format.
package httpapi

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/vardius/gorouter/v5"

	"github.com/vardius/go-api-boilerplate/internal/auth/app"
	"github.com/vardius/go-api-boilerplate/internal/auth/keys"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
	"github.com/vardius/go-api-boilerplate/internal/platform/run"
)

// Config is what the API serves with.
type Config struct {
	App      *app.Service
	Keys     *keys.Keys
	Probe    *run.Probe
	Verifier authn.TokenVerifier
	// Issuer is the public URL of this API, as the tokens' iss names it.
	Issuer string
	// WebURL is the web app, which hosts the sign-in, consent and OAuth
	// error pages.
	WebURL string
	// CookieSecure marks the cookies Secure; only plain-http development
	// turns it off.
	CookieSecure bool
	// RefreshCookiePath is the public path of /v1/sessions, the only place
	// the refresh cookie is sent.
	RefreshCookiePath string
	// TrustedProxies are the peers whose X-Forwarded-For names the client.
	TrustedProxies []netip.Prefix
	// RateLimit is how often one client may sign in, and, separately, call
	// the token and revocation endpoints.
	RateLimit httpx.RateLimitConfig
	Logger    *slog.Logger
}

// RefreshCookie is the cookie that holds a session's refresh token.
const RefreshCookie = "refresh_token"

// maxBody bounds every request body.
const maxBody = 64 << 10

// New returns the API's router.
func New(cfg Config) (*gorouter.Router, error) {
	issuer, err := url.Parse(cfg.Issuer)
	if err != nil || !issuer.IsAbs() || issuer.Host == "" {
		return nil, fmt.Errorf("httpapi: the issuer %q is not an absolute URL", cfg.Issuer)
	}
	if web, err := url.Parse(cfg.WebURL); err != nil || !web.IsAbs() || web.Host == "" {
		return nil, fmt.Errorf("httpapi: the web URL %q is not an absolute URL", cfg.WebURL)
	}
	if !strings.HasPrefix(cfg.RefreshCookiePath, "/") {
		return nil, fmt.Errorf("httpapi: the refresh cookie path %q is not absolute", cfg.RefreshCookiePath)
	}
	h := &handler{
		app:         cfg.App,
		verifier:    cfg.Verifier,
		issuer:      strings.TrimSuffix(cfg.Issuer, "/"),
		issuerPath:  strings.TrimSuffix(issuer.Path, "/"),
		web:         strings.TrimSuffix(cfg.WebURL, "/"),
		secure:      cfg.CookieSecure,
		refreshPath: cfg.RefreshCookiePath,
	}
	h.metadata = metadataFor(h.issuer)

	r := gorouter.New(
		gorouter.WithMiddleware(
			httpx.Recover(cfg.Logger),
			httpx.RequestID,
			httpx.Trace,
			httpx.Log(cfg.Logger),
			httpx.SecurityHeaders,
			httpx.BodyLimit(maxBody),
		),
		gorouter.WithNotFound(httpx.Handle(func(http.ResponseWriter, *http.Request) error {
			return apperr.New(apperr.ErrNotFound, "There is no such route.")
		})),
	)
	identify := authn.Authenticate(cfg.Verifier)
	firstParty := httpx.Chain(identify, authn.RequireFirstParty())
	signIns := httpx.RateLimit(cfg.RateLimit, cfg.TrustedProxies)
	clientCalls := httpx.RateLimit(cfg.RateLimit, cfg.TrustedProxies)
	cookie := httpx.CrossOrigin

	r.Handle(http.MethodGet, "/livez", cfg.Probe.Live())
	r.Handle(http.MethodGet, "/readyz", cfg.Probe.Ready())
	r.Handle(http.MethodGet, "/.well-known/jwks.json", cfg.Keys.JWKS())
	r.GET("/.well-known/oauth-authorization-server", h.serveMetadata)

	r.Handle(http.MethodPost, "/v1/sessions", httpx.Chain(signIns, cookie)(httpx.Handle(h.signIn)))
	r.Handle(http.MethodPost, "/v1/sessions/refresh", cookie(httpx.Handle(h.refresh)))
	r.Handle(http.MethodDelete, "/v1/sessions/current", cookie(httpx.Handle(h.signOut)))
	r.Handle(http.MethodGet, "/v1/sessions", firstParty(httpx.Handle(h.sessions)))
	r.Handle(http.MethodDelete, "/v1/sessions/{id}", httpx.Chain(cookie, firstParty)(httpx.Handle(h.revokeSession)))

	r.Handle(http.MethodGet, "/v1/clients", httpx.Chain(identify, authn.RequireUser(app.ScopeClientsRead))(httpx.Handle(h.clients)))
	r.Handle(http.MethodPost, "/v1/clients", httpx.Chain(cookie, identify, authn.RequireUser(app.ScopeClientsWrite))(httpx.Handle(h.createClient)))
	r.Handle(http.MethodDelete, "/v1/clients/{id}", httpx.Chain(cookie, identify, authn.RequireUser(app.ScopeClientsWrite))(httpx.Handle(h.deleteClient)))

	r.Handle(http.MethodGet, "/v1/authorization-requests/{id}", firstParty(httpx.Handle(h.authorizationRequest)))
	r.GET("/oauth/authorize", httpx.Handle(h.authorize))
	r.Handle(http.MethodPost, "/oauth/authorize", cookie(httpx.Handle(h.decide)))
	r.Handle(http.MethodPost, "/oauth/token", clientCalls(http.HandlerFunc(h.token)))
	r.Handle(http.MethodPost, "/oauth/revoke", clientCalls(http.HandlerFunc(h.revoke)))
	return r, nil
}

type handler struct {
	app         *app.Service
	verifier    authn.TokenVerifier
	issuer      string
	issuerPath  string
	web         string
	secure      bool
	refreshPath string
	metadata    []byte
}

// webPage returns the web app's page at path with query.
func (h *handler) webPage(path string, query url.Values) string {
	return h.web + path + "?" + query.Encode()
}

// setSession sets the cookies of a session's tokens.
func (h *handler) setSession(w http.ResponseWriter, t app.Tokens) {
	http.SetCookie(w, &http.Cookie{
		Name:     authn.AccessCookie,
		Value:    t.AccessToken.Reveal(),
		Path:     "/",
		MaxAge:   int(t.ExpiresIn / time.Second),
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshCookie,
		Value:    t.RefreshToken.Reveal(),
		Path:     h.refreshPath,
		MaxAge:   int(app.RefreshTokenTTL / time.Second),
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// clearSession expires both session cookies.
func (h *handler) clearSession(w http.ResponseWriter) {
	for _, c := range []*http.Cookie{
		{Name: authn.AccessCookie, Path: "/", SameSite: http.SameSiteLaxMode},
		{Name: RefreshCookie, Path: h.refreshPath, SameSite: http.SameSiteStrictMode},
	} {
		c.MaxAge, c.HttpOnly, c.Secure = -1, true, h.secure
		http.SetCookie(w, c)
	}
}

func refreshToken(r *http.Request) string {
	if c, err := r.Cookie(RefreshCookie); err == nil {
		return c.Value
	}
	return ""
}

// session returns the signed-in user of a browser request to an OAuth
// page: the identity of a valid first-party access cookie. Anything else
// is no session, so the user is asked to sign in rather than shown an
// error.
func (h *handler) session(r *http.Request) (authn.Identity, bool) {
	c, err := r.Cookie(authn.AccessCookie)
	if err != nil {
		return authn.Identity{}, false
	}
	id, err := h.verifier.Verify(r.Context(), c.Value)
	if err != nil || !id.IsUser() || !id.IsFirstParty() {
		return authn.Identity{}, false
	}
	return id, true
}
