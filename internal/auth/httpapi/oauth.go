package httpapi

import (
	"encoding/json/v2"
	"errors"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/auth/app"
	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/config"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
)

// metadata is the RFC 8414 authorization server metadata.
type metadata struct {
	Issuer                                 string   `json:"issuer"`
	AuthorizationEndpoint                  string   `json:"authorization_endpoint"`
	TokenEndpoint                          string   `json:"token_endpoint"`
	RevocationEndpoint                     string   `json:"revocation_endpoint"`
	JWKSURI                                string   `json:"jwks_uri"`
	ScopesSupported                        []string `json:"scopes_supported"`
	ResponseTypesSupported                 []string `json:"response_types_supported"`
	ResponseModesSupported                 []string `json:"response_modes_supported"`
	GrantTypesSupported                    []string `json:"grant_types_supported"`
	TokenEndpointAuthMethodsSupported      []string `json:"token_endpoint_auth_methods_supported"`
	RevocationEndpointAuthMethodsSupported []string `json:"revocation_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported          []string `json:"code_challenge_methods_supported"`
	AuthorizationResponseISSSupported      bool     `json:"authorization_response_iss_parameter_supported"`
}

func metadataFor(issuer string) []byte {
	auth := []string{"client_secret_basic", "client_secret_post"}
	body, err := json.Marshal(metadata{
		Issuer:                                 issuer,
		AuthorizationEndpoint:                  issuer + "/oauth/authorize",
		TokenEndpoint:                          issuer + "/oauth/token",
		RevocationEndpoint:                     issuer + "/oauth/revoke",
		JWKSURI:                                issuer + "/.well-known/jwks.json",
		ScopesSupported:                        app.GrantableScopes(),
		ResponseTypesSupported:                 []string{"code"},
		ResponseModesSupported:                 []string{"query"},
		GrantTypesSupported:                    []string{app.GrantAuthorizationCode, app.GrantClientCredentials, app.GrantRefreshToken},
		TokenEndpointAuthMethodsSupported:      auth,
		RevocationEndpointAuthMethodsSupported: auth,
		CodeChallengeMethodsSupported:          []string{"S256"},
		AuthorizationResponseISSSupported:      true,
	})
	if err != nil {
		panic(err) // strings and a bool always encode
	}
	return append(body, '\n')
}

func (h *handler) serveMetadata(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(h.metadata)
}

// authorize answers an authorization request (RFC 6749 §4.1.1): by sending
// the user agent to sign in, then to the consent page, or to the web app's
// error page. Never back to the client: only decide does that, once the
// user has answered.
func (h *handler) authorize(w http.ResponseWriter, r *http.Request) error {
	a, err := h.app.CheckAuthorization(r.Context(), r.URL.Query())
	if err != nil {
		return h.authorizeFailed(w, r, err, http.StatusFound)
	}
	id, ok := h.session(r)
	if !ok {
		next := h.issuerPath + "/oauth/authorize?" + r.URL.RawQuery
		http.Redirect(w, r, h.webPage("/sign-in", url.Values{"next": {next}}), http.StatusFound)
		return nil
	}
	request, err := h.app.RequestConsent(r.Context(), id.UserID, a)
	if err != nil {
		return err
	}
	http.Redirect(w, r, h.webPage("/consent", url.Values{"request": {request.String()}}), http.StatusFound)
	return nil
}

// authorizeFailed sends the user agent to the web app's error page with
// the reason of an *app.AuthorizeError, whose description goes only to the
// log.
func (h *handler) authorizeFailed(w http.ResponseWriter, r *http.Request, err error, status int) error {
	ae, ok := errors.AsType[*app.AuthorizeError](err)
	if !ok {
		return err
	}
	httpx.RecordError(w, err)
	http.Redirect(w, r, h.webPage("/oauth/error", url.Values{"reason": {ae.Code}}), status)
	return nil
}

// decide takes the consent page's form: request, the pending request's
// ID, and decision, allow or deny.
func (h *handler) decide(w http.ResponseWriter, r *http.Request) error {
	fail := func(reason, description string) error {
		return h.authorizeFailed(w, r, &app.AuthorizeError{Code: reason, Description: description}, http.StatusSeeOther)
	}
	id, ok := h.session(r)
	if !ok {
		return fail(app.ReasonLoginRequired, "The consent form came without a session.")
	}
	if err := r.ParseForm(); err != nil {
		if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
			return err
		}
		return fail(app.ReasonInvalidRequest, "The consent form doesn't parse.")
	}
	var allow bool
	switch d := r.PostForm["decision"]; {
	case len(d) == 1 && d[0] == "allow":
		allow = true
	case len(d) == 1 && d[0] == "deny":
	default:
		return fail(app.ReasonInvalidRequest, "The decision must be allow or deny.")
	}
	reqs := r.PostForm["request"]
	request, err := uuid.Parse(strings.Join(reqs, ""))
	if len(reqs) != 1 || err != nil {
		return fail(app.ReasonInvalidRequest, "The consent form names no request.")
	}
	loc, err := h.app.Decide(r.Context(), id.UserID, request, allow)
	if err != nil {
		return h.authorizeFailed(w, r, err, http.StatusSeeOther)
	}
	http.Redirect(w, r, loc, http.StatusSeeOther)
	return nil
}

type consentJSON struct {
	Client struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"client"`
	Scopes       []scopeJSON `json:"scopes"`
	RedirectHost string      `json:"redirect_host"`
}

type scopeJSON struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

var errNoRequest = apperr.New(apperr.ErrNotFound, "No such authorization request.")

func (h *handler) authorizationRequest(w http.ResponseWriter, r *http.Request) error {
	id, _ := authn.FromContext(r.Context())
	request, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return errNoRequest
	}
	c, err := h.app.Consent(r.Context(), id.UserID, request)
	if err != nil {
		return err
	}
	var out consentJSON
	out.Client.ID, out.Client.Name = c.ClientID, c.ClientName
	out.Scopes = make([]scopeJSON, len(c.Scopes))
	for i, s := range c.Scopes {
		out.Scopes[i] = scopeJSON(s)
	}
	if u, err := url.Parse(c.RedirectURI); err == nil {
		out.RedirectHost = u.Host
	}
	return httpx.WriteJSON(w, http.StatusOK, out)
}

type tokenJSON struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitzero"`
	Scope        string `json:"scope"`
}

func (h *handler) token(w http.ResponseWriter, r *http.Request) {
	c, form, err := h.authenticateClient(r)
	if err != nil {
		h.oauthFailed(w, r, err)
		return
	}
	t, err := h.app.Token(r.Context(), c, form)
	if err != nil {
		h.oauthFailed(w, r, err)
		return
	}
	noStore(w)
	if err := httpx.WriteJSON(w, http.StatusOK, tokenJSON{
		AccessToken:  t.AccessToken.Reveal(),
		TokenType:    "Bearer",
		ExpiresIn:    int(t.ExpiresIn / time.Second),
		RefreshToken: t.RefreshToken.Reveal(),
		Scope:        strings.Join(t.Scopes, " "),
	}); err != nil {
		httpx.WriteError(w, r, err)
	}
}

func (h *handler) revoke(w http.ResponseWriter, r *http.Request) {
	c, form, err := h.authenticateClient(r)
	if err == nil {
		err = h.app.Revoke(r.Context(), c, form)
	}
	if err != nil {
		h.oauthFailed(w, r, err)
		return
	}
	noStore(w)
	w.WriteHeader(http.StatusOK)
}

// noStore keeps a token response out of every cache, RFC 6749 §5.1.
func noStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}

// authenticateClient reads the form of a token or revocation request and
// authenticates the client that sent it, by HTTP Basic (RFC 6749 §2.3.1,
// with both parts form-encoded) or by client_id and client_secret in the
// form. Using both, or neither, is refused.
func (h *handler) authenticateClient(r *http.Request) (store.Client, url.Values, error) {
	ct, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || ct != "application/x-www-form-urlencoded" {
		return store.Client{}, nil, oauthError(app.ErrorInvalidRequest, "The body must be application/x-www-form-urlencoded.")
	}
	if err := r.ParseForm(); err != nil {
		if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
			return store.Client{}, nil, err
		}
		return store.Client{}, nil, oauthError(app.ErrorInvalidRequest, "The body doesn't parse as a form.")
	}
	form := r.PostForm
	for _, p := range []string{"client_id", "client_secret"} {
		if len(form[p]) > 1 {
			return store.Client{}, nil, oauthError(app.ErrorInvalidRequest, "The "+p+" parameter is repeated.")
		}
	}

	id, secret := form.Get("client_id"), form.Get("client_secret")
	if header := r.Header.Get("Authorization"); header != "" {
		user, pass, ok := r.BasicAuth()
		if ok {
			user, err = url.QueryUnescape(user)
			if err == nil {
				pass, err = url.QueryUnescape(pass)
			}
			ok = err == nil
		}
		switch {
		case !ok:
			return store.Client{}, nil, oauthError(app.ErrorInvalidClient, "The Authorization header must hold HTTP Basic client credentials.")
		case secret != "":
			return store.Client{}, nil, oauthError(app.ErrorInvalidRequest, "Authenticate the client one way: HTTP Basic or the form, not both.")
		case id != "" && id != user:
			return store.Client{}, nil, oauthError(app.ErrorInvalidClient, "The client_id doesn't match the authenticated client.")
		}
		id, secret = user, pass
	}
	c, err := h.app.AuthenticateClient(r.Context(), id, config.Secret(secret))
	return c, form, err
}

type oauthErrorJSON struct {
	Error       string `json:"error"`
	Description string `json:"error_description,omitzero"`
}

func oauthError(code, description string) *app.OAuthError {
	return &app.OAuthError{Code: code, Description: description}
}

// oauthFailed answers a failed token or revocation request: an
// *app.OAuthError in the format of RFC 6749 §5.2, anything else as a
// problem.
func (h *handler) oauthFailed(w http.ResponseWriter, r *http.Request, err error) {
	oe, ok := errors.AsType[*app.OAuthError](err)
	if !ok {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.RecordError(w, err)
	status := http.StatusBadRequest
	if oe.Code == app.ErrorInvalidClient {
		status = http.StatusUnauthorized
		w.Header().Set("WWW-Authenticate", `Basic realm="oauth", charset="UTF-8"`)
	}
	noStore(w)
	if err := httpx.WriteJSON(w, status, oauthErrorJSON{Error: oe.Code, Description: oe.Description}); err != nil {
		httpx.WriteError(w, r, err)
	}
}
