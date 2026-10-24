package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"
	"uuid"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/config"
	"github.com/vardius/go-api-boilerplate/internal/user/app"
)

// GoogleConfig turns on sign-in with Google. Services nest it with
// envPrefix:"GOOGLE_".
type GoogleConfig struct {
	ClientID     string        `env:"CLIENT_ID" desc:"OAuth client ID of the Google Cloud app. Google sign-in is off while it is empty."`
	ClientSecret config.Secret `env:"CLIENT_SECRET" desc:"That OAuth client's secret. Required with GOOGLE_CLIENT_ID."`
	RedirectURL  string        `env:"REDIRECT_URL" desc:"The redirect URI registered with Google: the public URL of /v1/google/callback. Required with GOOGLE_CLIENT_ID."`
}

// Enabled reports whether c turns Google sign-in on.
func (c GoogleConfig) Enabled() bool { return c.ClientID != "" }

// GoogleIssuer is Google's OpenID Connect issuer.
const GoogleIssuer = "https://accounts.google.com"

// flowTTL is how long a Google sign-in may take, from start to callback.
const flowTTL = 10 * time.Minute

// flowCookie holds one sign-in's state, nonce and PKCE verifier between
// /v1/google/start and the callback.
const flowCookie = "google_flow"

// Google signs users in with Google over OpenID Connect: the authorization
// code flow with PKCE S256, a state and a nonce.
type Google struct {
	// Now is the clock ID tokens are checked against. It defaults to
	// time.Now.
	Now func() time.Time

	clientID     string
	clientSecret config.Secret
	redirectURL  string
	issuer       string
	client       *http.Client
	// cookiePath and secure scope the flow cookie to the callback's
	// directory, over https when the callback is.
	cookiePath string
	secure     bool

	mu       sync.Mutex
	oauth    *oauth2.Config // nil until discovered
	verifier *oidc.IDTokenVerifier
}

// NewGoogle returns a Google signing in through issuer, normally
// GoogleIssuer, with requests to it made by client. Google's endpoints are
// discovered on first use, so the service starts whether or not Google is
// reachable.
func NewGoogle(cfg GoogleConfig, issuer string, client *http.Client) (*Google, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, errors.New("google: GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET are both required")
	}
	redirect, err := url.Parse(cfg.RedirectURL)
	if err != nil || (redirect.Scheme != "https" && redirect.Scheme != "http") || redirect.Host == "" || !strings.HasSuffix(redirect.Path, callbackPath) {
		return nil, fmt.Errorf("google: GOOGLE_REDIRECT_URL %q is not an http or https URL ending in %s", cfg.RedirectURL, callbackPath)
	}
	return &Google{
		Now:          time.Now,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		redirectURL:  cfg.RedirectURL,
		issuer:       issuer,
		client:       client,
		cookiePath:   path.Dir(redirect.EscapedPath()),
		secure:       redirect.Scheme == "https",
	}, nil
}

// discover fetches the issuer's endpoints and keys once; a failure is
// retried on the next sign-in.
func (g *Google) discover(ctx context.Context) (*oauth2.Config, *oidc.IDTokenVerifier, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.oauth == nil {
		p, err := oidc.NewProvider(oidc.ClientContext(ctx, g.client), g.issuer)
		if err != nil {
			return nil, nil, fmt.Errorf("discover google: %w", err)
		}
		endpoint := p.Endpoint()
		// client_secret_basic, which Google takes; detecting it would
		// send every failed exchange twice.
		endpoint.AuthStyle = oauth2.AuthStyleInHeader
		g.oauth = &oauth2.Config{
			ClientID:     g.clientID,
			ClientSecret: g.clientSecret.Reveal(),
			Endpoint:     endpoint,
			RedirectURL:  g.redirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "email"},
		}
		g.verifier = p.Verifier(&oidc.Config{ClientID: g.clientID, Now: func() time.Time { return g.Now() }})
	}
	return g.oauth, g.verifier, nil
}

// flow is one sign-in in progress, kept in the browser between start and
// callback. Nothing in it is a credential on its own: the verifier is
// useless without Google's code, which goes only to the redirect URI.
type flow struct {
	State    string `json:"state"`
	Nonce    string `json:"nonce"`
	Verifier string `json:"verifier"`
	Next     string `json:"next,omitzero"`
	// SignInState is the web app's, for its sign-in link; not Google's State.
	SignInState string    `json:"sign_in_state,omitzero"`
	LinkTo      uuid.UUID `json:"link_to,omitzero"` // the user linking Google; zero for a sign-in
	Started     time.Time `json:"started"`
}

func (f flow) linking() bool { return f.LinkTo != uuid.Nil() }

func newFlow(next, signInState string, linkTo uuid.UUID, now time.Time) flow {
	return flow{
		State:       rand.Text(),
		Nonce:       rand.Text(),
		Verifier:    oauth2.GenerateVerifier(),
		Next:        app.SafeNext(next),
		SignInState: app.SafeSignInState(signInState),
		LinkTo:      linkTo,
		Started:     now,
	}
}

// begin returns Google's consent page for f and the cookie that keeps f.
func (g *Google) begin(ctx context.Context, f flow) (string, *http.Cookie, error) {
	oauth, _, err := g.discover(ctx)
	if err != nil {
		return "", nil, err
	}
	value, err := json.Marshal(f)
	if err != nil {
		return "", nil, fmt.Errorf("encode google flow: %w", err)
	}
	to := oauth.AuthCodeURL(f.State, oauth2.S256ChallengeOption(f.Verifier), oidc.Nonce(f.Nonce))
	return to, g.cookie(base64.RawURLEncoding.EncodeToString(value), int(flowTTL/time.Second)), nil
}

func (g *Google) cookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     flowCookie,
		Value:    value,
		Path:     g.cookiePath,
		MaxAge:   maxAge,
		Secure:   g.secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode, // sent on Google's top-level redirect back
	}
}

var errFlowExpired = errors.New("google sign-in state is missing, expired or not this one")

// resume returns the flow the callback r belongs to and clears its cookie,
// which works once.
func (g *Google) resume(w http.ResponseWriter, r *http.Request, now time.Time) (flow, error) {
	c, err := r.Cookie(flowCookie)
	if err != nil {
		return flow{}, errFlowExpired
	}
	http.SetCookie(w, g.cookie("", -1))
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		return flow{}, errFlowExpired
	}
	var f flow
	if err := json.Unmarshal(raw, &f); err != nil {
		return flow{}, errFlowExpired
	}
	state := r.URL.Query().Get("state")
	if f.State == "" || subtle.ConstantTimeCompare([]byte(state), []byte(f.State)) != 1 || !now.Before(f.Started.Add(flowTTL)) {
		return f, errFlowExpired
	}
	return f, nil
}

// account exchanges Google's code for an ID token and returns who it says
// signed in.
func (g *Google) account(ctx context.Context, code string, f flow) (app.GoogleAccount, error) {
	oauth, verifier, err := g.discover(ctx)
	if err != nil {
		return app.GoogleAccount{}, err
	}
	token, err := oauth.Exchange(oidc.ClientContext(ctx, g.client), code, oauth2.VerifierOption(f.Verifier))
	if err != nil {
		return app.GoogleAccount{}, fmt.Errorf("exchange google code: %w", err)
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		return app.GoogleAccount{}, errors.New("google token response has no id_token")
	}
	idToken, err := verifier.Verify(ctx, raw)
	if err != nil {
		return app.GoogleAccount{}, fmt.Errorf("verify google id token: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(idToken.Nonce), []byte(f.Nonce)) != 1 {
		return app.GoogleAccount{}, errors.New("google id token nonce is not this sign-in's")
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return app.GoogleAccount{}, fmt.Errorf("read google id token claims: %w", err)
	}
	return app.GoogleAccount{Subject: idToken.Subject, Email: claims.Email, EmailVerified: claims.EmailVerified}, nil
}

var errGoogleDenied = errors.New("the user declined at google")

// outcome is what the web app is told about a Google sign-in that failed
// with err.
func outcome(err error) string {
	switch {
	case errors.Is(err, errGoogleDenied):
		return app.GoogleDenied
	case errors.Is(err, errFlowExpired), errors.Is(err, apperr.ErrUnauthorized):
		return app.GoogleExpired
	case errors.Is(err, apperr.ErrForbidden):
		return app.GoogleUnverified
	case errors.Is(err, apperr.ErrConflict):
		return app.GoogleTaken
	default:
		return app.GoogleFailed
	}
}
