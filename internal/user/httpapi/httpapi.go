// Package httpapi is the user service's public HTTP API, the routes the
// gateway serves under /api/user. api/user.openapi.yaml documents every
// route, and a test holds the two to each other.
//
// Who may call a route is decided by the use case, from the authn.Identity
// that authn.Authenticate puts in the context; the routes that take no
// identity don't authenticate at all, so a stale cookie can't fail them.
package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
	"time"
	"uuid"

	"github.com/vardius/gorouter/v5"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
	"github.com/vardius/go-api-boilerplate/internal/platform/run"
	"github.com/vardius/go-api-boilerplate/internal/user/app"
	"github.com/vardius/go-api-boilerplate/internal/user/domain"
	"github.com/vardius/go-api-boilerplate/internal/user/store"
)

// Config is what the API serves with.
type Config struct {
	App      *app.Service
	Probe    *run.Probe
	Verifier authn.TokenVerifier
	// Google, when set, turns on Google sign-in.
	Google *Google
	// MailPreviewURL, when set, is shown by the web app next to "check your
	// inbox": a local mail catcher such as Mailpit.
	MailPreviewURL string
	// TrustedProxies are the peers whose X-Forwarded-For names the client.
	TrustedProxies []netip.Prefix
	// RateLimit is how often one client may ask for login links.
	RateLimit httpx.RateLimitConfig
	Logger    *slog.Logger
	// Now is the clock. It defaults to time.Now.
	Now func() time.Time
}

// maxBody bounds every JSON request body.
const maxBody = 4 << 10

// perEmail is how many login links one address may be sent: five at once,
// then one a minute.
var perEmail = httpx.RateLimitConfig{Rate: 1.0 / 60, Burst: 5, Visitors: 10000}

// New returns the API's router.
func New(cfg Config) *gorouter.Router {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	h := &handler{
		app:      cfg.App,
		google:   cfg.Google,
		preview:  cfg.MailPreviewURL,
		perEmail: httpx.NewLimiter[string](perEmail),
		log:      cfg.Logger,
		now:      cfg.Now,
	}
	r := gorouter.New(
		gorouter.WithMiddleware(
			httpx.Recover(cfg.Logger),
			httpx.RequestID,
			httpx.Trace,
			httpx.Log(cfg.Logger),
			httpx.SecurityHeaders,
			httpx.CrossOrigin,
		),
		gorouter.WithNotFound(httpx.Handle(func(http.ResponseWriter, *http.Request) error {
			return apperr.New(apperr.ErrNotFound, "There is no such route.")
		})),
	)
	identify := authn.Authenticate(cfg.Verifier)
	perIP := httpx.RateLimit(cfg.RateLimit, cfg.TrustedProxies)

	r.Handle(http.MethodGet, "/livez", cfg.Probe.Live())
	r.Handle(http.MethodGet, "/readyz", cfg.Probe.Ready())
	r.GET("/v1/config", httpx.Handle(h.config))
	r.Handle(http.MethodPost, "/v1/login-links", perIP(httpx.Handle(h.requestLoginLink)))
	r.Handle(http.MethodGet, "/v1/me", identify(httpx.Handle(h.me)))
	r.Handle(http.MethodPost, "/v1/me/email-change", identify(httpx.Handle(h.requestEmailChange)))
	r.Handle(http.MethodPost, "/v1/me/email-change/confirm", identify(httpx.Handle(h.confirmEmailChange)))
	r.Handle(http.MethodGet, "/v1/google/start", identify(httpx.Handle(h.googleStart)))
	r.Handle(http.MethodGet, callbackPath, identify(http.HandlerFunc(h.googleCallback)))
	return r
}

// callbackPath is Google's redirect URI on this service.
const callbackPath = "/v1/google/callback"

type handler struct {
	app      *app.Service
	google   *Google
	preview  string
	perEmail *httpx.Limiter[string]
	log      *slog.Logger
	now      func() time.Time
}

type configJSON struct {
	GoogleEnabled  bool   `json:"google_enabled"`
	MailPreviewURL string `json:"mail_preview_url,omitzero"`
}

func (h *handler) config(w http.ResponseWriter, _ *http.Request) error {
	return httpx.WriteJSON(w, http.StatusOK, configJSON{GoogleEnabled: h.google != nil, MailPreviewURL: h.preview})
}

type userJSON struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	GoogleLinked bool      `json:"google_linked"`
	CreatedAt    time.Time `json:"created_at"`
}

func toJSON(u store.User) userJSON {
	return userJSON{ID: u.ID, Email: u.Email, GoogleLinked: u.GoogleSubject != "", CreatedAt: u.CreatedAt}
}

// accepted is the body of a 202: the work is queued, and there is nothing
// else to say.
type accepted struct{}

func (h *handler) requestLoginLink(w http.ResponseWriter, r *http.Request) error {
	body, err := httpx.DecodeJSON[struct {
		Email string `json:"email"`
		Next  string `json:"next"`
		State string `json:"state"`
	}](r, maxBody)
	if err != nil {
		return err
	}
	addr, err := domain.ParseEmail(body.Email)
	if err != nil {
		return err
	}
	// Limited after parsing, so one address is one bucket however it's
	// spelled. Every address gets one, account or not.
	if err := h.perEmail.Allow(addr.String(), h.now()); err != nil {
		return err
	}
	if err := h.app.RequestLoginLink(r.Context(), addr.String(), body.Next, body.State); err != nil {
		return err
	}
	return httpx.WriteJSON(w, http.StatusAccepted, accepted{})
}

func (h *handler) me(w http.ResponseWriter, r *http.Request) error {
	u, err := h.app.Me(r.Context())
	if err != nil {
		return err
	}
	return httpx.WriteJSON(w, http.StatusOK, toJSON(u))
}

func (h *handler) requestEmailChange(w http.ResponseWriter, r *http.Request) error {
	body, err := httpx.DecodeJSON[struct {
		Email string `json:"email"`
	}](r, maxBody)
	if err != nil {
		return err
	}
	if err := h.app.RequestEmailChange(r.Context(), body.Email); err != nil {
		return err
	}
	return httpx.WriteJSON(w, http.StatusAccepted, accepted{})
}

func (h *handler) confirmEmailChange(w http.ResponseWriter, r *http.Request) error {
	body, err := httpx.DecodeJSON[struct {
		Code string `json:"code"`
	}](r, maxBody)
	if err != nil {
		return err
	}
	u, err := h.app.ConfirmEmailChange(r.Context(), body.Code)
	if err != nil {
		return err
	}
	return httpx.WriteJSON(w, http.StatusOK, toJSON(u))
}

var errGoogleOff = apperr.New(apperr.ErrNotFound, "Google sign-in is not set up on this server.")

// googleStart sends the browser to Google: to sign in, or with ?link=1 to
// link a Google account to the signed-in user.
func (h *handler) googleStart(w http.ResponseWriter, r *http.Request) error {
	if h.google == nil {
		return errGoogleOff
	}
	q := r.URL.Query()
	linkTo := uuid.Nil()
	switch q.Get("link") {
	case "":
	case "1":
		id, err := app.Actor(r.Context(), app.ScopeUserWrite)
		if err != nil {
			return err
		}
		linkTo = id
	default:
		return apperr.Invalid("link", "must be 1 or absent")
	}
	to, cookie, err := h.google.begin(r.Context(), newFlow(q.Get("next"), q.Get("state"), linkTo, h.now()))
	if err != nil {
		return err
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, to, http.StatusFound)
	return nil
}

// googleCallback finishes what googleStart began and sends the browser back
// to the web app: signed in, linked, or to a page saying what went wrong.
func (h *handler) googleCallback(w http.ResponseWriter, r *http.Request) {
	if h.google == nil {
		httpx.WriteError(w, r, errGoogleOff)
		return
	}
	f, err := h.google.resume(w, r, h.now())
	var to *url.URL
	if err == nil {
		to, err = h.finishGoogle(r.Context(), r.URL.Query(), f)
	}
	if err != nil {
		reason := outcome(err)
		if reason == app.GoogleFailed {
			h.log.ErrorContext(r.Context(), "google sign-in failed", "err", err)
		}
		to = h.app.GoogleFailurePage(f.linking(), reason)
	}
	http.Redirect(w, r, to.String(), http.StatusFound)
}

func (h *handler) finishGoogle(ctx context.Context, q url.Values, f flow) (*url.URL, error) {
	switch e := q.Get("error"); e {
	case "":
	case "access_denied":
		return nil, errGoogleDenied
	default:
		return nil, fmt.Errorf("google answered %q", e)
	}
	account, err := h.google.account(ctx, q.Get("code"), f)
	if err != nil {
		return nil, err
	}
	if !f.linking() {
		return h.app.GoogleSignIn(ctx, account, f.Next, f.SignInState)
	}
	// Linking needs the same user at both ends of the round trip.
	if id, err := app.Actor(ctx, app.ScopeUserWrite); err != nil || id != f.LinkTo {
		return nil, errFlowExpired
	}
	return h.app.LinkGoogle(ctx, account.Subject)
}
