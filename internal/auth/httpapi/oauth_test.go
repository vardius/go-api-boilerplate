package httpapi_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/vardius/go-api-boilerplate/internal/auth/app/apptest"
	"github.com/vardius/go-api-boilerplate/internal/auth/httpapi"
	"github.com/vardius/go-api-boilerplate/internal/auth/keys"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/run"
)

// consent takes the browser from an authorization URL through the consent
// page to where the decision sends it.
func (r *rig) consent(t *testing.T, b *browser, authorizeURL, decision string) *url.URL {
	t.Helper()
	loc := b.expect(http.MethodGet, strings.TrimPrefix(authorizeURL, r.srv.URL), nil, http.StatusFound).location(t)
	if !strings.HasPrefix(loc.String(), webURL+"/consent?") || loc.Query().Get("request") == "" {
		t.Fatalf("sent to %s, want the consent page", loc)
	}
	request := loc.Query().Get("request")
	var details struct {
		Client struct{ ID, Name string } `json:"client"`
		Scopes []struct {
			Name, Description string
		} `json:"scopes"`
		RedirectHost string `json:"redirect_host"`
	}
	b.expect(http.MethodGet, "/v1/authorization-requests/"+request, nil, http.StatusOK).decode(t, &details)
	if details.Client.Name != "Test app" || len(details.Scopes) == 0 || details.Scopes[0].Description == "" || details.RedirectHost != "localhost:9999" {
		t.Fatalf("consent details = %+v", details)
	}
	res := b.form("/oauth/authorize", url.Values{"request": {request}, "decision": {decision}}, nil)
	if res.status != http.StatusSeeOther {
		t.Fatalf("POST /oauth/authorize = %d: %s", res.status, res.body)
	}
	return res.location(t)
}

func (r *rig) oauthConfig(c clientJSON, secret string, scopes ...string) oauth2.Config {
	return oauth2.Config{
		ClientID:     c.ID,
		ClientSecret: secret,
		RedirectURL:  appRedirect,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:   r.srv.URL + "/oauth/authorize",
			TokenURL:  r.srv.URL + "/oauth/token",
			AuthStyle: oauth2.AuthStyleInHeader,
		},
	}
}

// The whole flow, with golang.org/x/oauth2 as the client.
func TestOAuthFlowWithGolangOAuth2(t *testing.T) {
	r := newRig(t, options{})
	user := uuid.NewV7()
	b := r.signIn(t, user)
	c, secret := createClient(t, b, "user:read", "clients:read")
	ctx := context.WithValue(t.Context(), oauth2.HTTPClient, r.httpClient(t, nil))
	cfg := r.oauthConfig(c, secret, "user:read")
	verifier := oauth2.GenerateVerifier()

	back := r.consent(t, b, cfg.AuthCodeURL("the-state", oauth2.S256ChallengeOption(verifier)), "allow")
	if q := back.Query(); !strings.HasPrefix(back.String(), appRedirect+"?") || q.Get("state") != "the-state" || q.Get("iss") != apptest.Issuer || q.Get("code") == "" {
		t.Fatalf("sent back to %s", back)
	}
	code := back.Query().Get("code")

	token, err := cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	id, err := r.e.Verifier().Verify(t.Context(), token.AccessToken)
	if err != nil || id.UserID != user || id.ClientID != c.ID || strings.Join(id.Scopes, " ") != "user:read" {
		t.Fatalf("access token for %+v (%v)", id, err)
	}
	if token.TokenType != "Bearer" || token.RefreshToken == "" || token.Extra("scope") != "user:read" || time.Until(token.Expiry) > 15*time.Minute {
		t.Fatalf("token = %+v", token)
	}

	expired := *token
	expired.Expiry = time.Now().Add(-time.Minute)
	fresh, err := cfg.TokenSource(ctx, &expired).Token()
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if fresh.RefreshToken == token.RefreshToken || fresh.AccessToken == token.AccessToken {
		t.Fatal("refresh didn't rotate")
	}

	t.Run("revoking the refresh token ends the grant", func(t *testing.T) {
		app := r.browser(t)
		res := app.form("/oauth/revoke", url.Values{"token": {fresh.RefreshToken}, "token_type_hint": {"refresh_token"}}, basic(c.ID, secret))
		if res.status != http.StatusOK || len(res.body) != 0 {
			t.Fatalf("revoke = %d: %s", res.status, res.body)
		}
		stale := *fresh
		stale.Expiry = time.Now().Add(-time.Minute)
		if _, err := cfg.TokenSource(ctx, &stale).Token(); err == nil {
			t.Fatal("a revoked refresh token worked")
		}
	})

	t.Run("client credentials, sent in the form", func(t *testing.T) {
		cc := clientcredentials.Config{ClientID: c.ID, ClientSecret: secret, TokenURL: cfg.Endpoint.TokenURL, AuthStyle: oauth2.AuthStyleInParams}
		token, err := cc.Token(ctx)
		if err != nil {
			t.Fatal(err)
		}
		id, err := r.e.Verifier().Verify(t.Context(), token.AccessToken)
		if err != nil || id.IsUser() || id.Subject != c.ID || strings.Join(id.Scopes, " ") != "user:read clients:read" || token.RefreshToken != "" {
			t.Fatalf("client token for %+v (%v), %+v", id, err, token)
		}
		cc.Scopes = []string{"clients:write"}
		if _, err := cc.Token(ctx); err == nil || !strings.Contains(err.Error(), "invalid_scope") {
			t.Fatalf("a scope wider than the client's: %v", err)
		}
	})

	t.Run("the logs hold no token material", func(t *testing.T) {
		logs := r.logs.String()
		for _, secret := range []string{secret, code, verifier, token.AccessToken, token.RefreshToken, fresh.AccessToken, fresh.RefreshToken} {
			if strings.Contains(logs, secret) {
				t.Fatalf("the logs hold a secret:\n%s", logs)
			}
		}
		if !strings.Contains(logs, "invalid_grant") || !strings.Contains(logs, "invalid_scope") {
			t.Fatalf("the logs don't say why requests failed:\n%s", logs)
		}
	})
}

func basic(id, secret string) http.Header {
	req := &http.Request{Header: http.Header{}}
	req.SetBasicAuth(url.QueryEscape(id), url.QueryEscape(secret))
	return req.Header
}

func TestAuthorizationCodeRules(t *testing.T) {
	r := newRig(t, options{})
	b := r.signIn(t, uuid.NewV7())
	c, secret := createClient(t, b, "user:read")
	ctx := context.WithValue(t.Context(), oauth2.HTTPClient, r.httpClient(t, nil))
	cfg := r.oauthConfig(c, secret)
	newCode := func(verifier string) string {
		return r.consent(t, b, cfg.AuthCodeURL("s", oauth2.S256ChallengeOption(verifier)), "allow").Query().Get("code")
	}
	exchange := func(form url.Values) response {
		form.Set("grant_type", "authorization_code")
		form.Set("redirect_uri", appRedirect)
		return r.browser(t).form("/oauth/token", form, basic(c.ID, secret))
	}

	t.Run("a missing or wrong verifier is invalid_grant and spends the code", func(t *testing.T) {
		for _, wrong := range []url.Values{{}, {"code_verifier": {oauth2.GenerateVerifier()}}} {
			verifier := oauth2.GenerateVerifier()
			code := newCode(verifier)
			wrong.Set("code", code)
			if res := exchange(wrong); res.status != http.StatusBadRequest || res.oauthError(t) != "invalid_grant" {
				t.Fatalf("exchange = %d %s", res.status, res.body)
			}
			if res := exchange(url.Values{"code": {code}, "code_verifier": {verifier}}); res.oauthError(t) != "invalid_grant" {
				t.Fatalf("the code survived a failed exchange: %s", res.body)
			}
		}
	})

	t.Run("a code used twice fails and revokes what the first use issued", func(t *testing.T) {
		verifier := oauth2.GenerateVerifier()
		code := newCode(verifier)
		first, err := cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
		if err != nil {
			t.Fatal(err)
		}
		if res := exchange(url.Values{"code": {code}, "code_verifier": {verifier}}); res.oauthError(t) != "invalid_grant" {
			t.Fatalf("second use = %s", res.body)
		}
		res := r.browser(t).form("/oauth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {first.RefreshToken}}, basic(c.ID, secret))
		if res.oauthError(t) != "invalid_grant" {
			t.Fatalf("the first use's refresh token still works: %s", res.body)
		}
	})

	t.Run("refresh token reuse revokes the grant", func(t *testing.T) {
		verifier := oauth2.GenerateVerifier()
		first, err := cfg.Exchange(ctx, newCode(verifier), oauth2.VerifierOption(verifier))
		if err != nil {
			t.Fatal(err)
		}
		refresh := func(token string) response {
			return r.browser(t).form("/oauth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {token}}, basic(c.ID, secret))
		}
		var second struct {
			RefreshToken string `json:"refresh_token"`
		}
		res := refresh(first.RefreshToken)
		if res.status != http.StatusOK {
			t.Fatalf("refresh = %s", res.body)
		}
		res.decode(t, &second)
		if res := refresh(first.RefreshToken); res.oauthError(t) != "invalid_grant" {
			t.Fatalf("reuse = %s", res.body)
		}
		if res := refresh(second.RefreshToken); res.oauthError(t) != "invalid_grant" {
			t.Fatalf("the grant survived reuse: %s", res.body)
		}
	})
}

func TestAuthorizeRedirects(t *testing.T) {
	r := newRig(t, options{})
	user := uuid.NewV7()
	b := r.signIn(t, user)
	c, _ := createClient(t, b, "user:read")
	query := func(edit func(url.Values)) string {
		q := url.Values{
			"response_type":         {"code"},
			"client_id":             {c.ID},
			"redirect_uri":          {appRedirect},
			"state":                 {"st"},
			"code_challenge":        {oauth2.S256ChallengeFromVerifier(oauth2.GenerateVerifier())},
			"code_challenge_method": {"S256"},
		}
		edit(q)
		return "/oauth/authorize?" + q.Encode()
	}

	// A bad request goes to the error page, signed in or not, and never
	// back to the client, not even to a redirect URI it registered: anyone
	// can register a client, so that would make this an open redirector
	// (RFC 9700 §4.11.2). The client hears only the answer of a user who
	// saw the consent page.
	anonymous := r.browser(t)
	for name, tt := range map[string]struct {
		edit   func(url.Values)
		reason string
	}{
		"an unknown client":               {func(q url.Values) { q.Set("client_id", uuid.NewV4().String()) }, "invalid_client"},
		"an unregistered redirect":        {func(q url.Values) { q.Set("redirect_uri", "https://evil.example/cb") }, "invalid_redirect_uri"},
		"a redirect with a parameter":     {func(q url.Values) { q.Set("redirect_uri", appRedirect+"?evil=1") }, "invalid_redirect_uri"},
		"a repeated parameter":            {func(q url.Values) { q.Add("state", "other") }, "invalid_request"},
		"no response_type":                {func(q url.Values) { q.Del("response_type") }, "invalid_request"},
		"the implicit grant":              {func(q url.Values) { q.Set("response_type", "token") }, "unsupported_response_type"},
		"no PKCE":                         {func(q url.Values) { q.Del("code_challenge"); q.Del("code_challenge_method") }, "pkce_required"},
		"PKCE plain":                      {func(q url.Values) { q.Set("code_challenge_method", "plain") }, "pkce_required"},
		"a scope wider than the client's": {func(q url.Values) { q.Set("scope", "user:read user:write") }, "invalid_scope"},
	} {
		for who, from := range map[string]*browser{"signed in": b, "anonymous": anonymous} {
			t.Run(name+", "+who, func(t *testing.T) {
				res := from.expect(http.MethodGet, query(tt.edit), nil, http.StatusFound)
				if loc, want := res.header.Get("Location"), webURL+"/oauth/error?reason="+tt.reason; loc != want {
					t.Fatalf("sent to %s, want %s", loc, want)
				}
			})
		}
	}

	t.Run("a client registered by an attacker sends no one to them", func(t *testing.T) {
		const landing = "https://evil.example/landing"
		var created struct {
			Client clientJSON `json:"client"`
		}
		r.signIn(t, uuid.NewV7()).expect(http.MethodPost, "/v1/clients", map[string]any{
			"name": "Totally legit", "redirect_uris": []string{landing}, "scopes": []string{"user:read"},
		}, http.StatusCreated).decode(t, &created)
		q := url.Values{"client_id": {created.Client.ID}, "redirect_uri": {landing}, "response_type": {"code"}}
		loc := r.browser(t).expect(http.MethodGet, "/oauth/authorize?"+q.Encode(), nil, http.StatusFound).header.Get("Location")
		if loc != webURL+"/oauth/error?reason=pkce_required" {
			t.Fatalf("a visitor is sent to %s", loc)
		}
	})

	t.Run("the log says what was wrong", func(t *testing.T) {
		if logs := r.logs.String(); !strings.Contains(logs, "The code_challenge_method must be S256.") {
			t.Fatalf("the logs don't say why the request failed:\n%s", logs)
		}
	})

	path := query(func(url.Values) {})
	wantSignIn := webURL + "/sign-in?" + url.Values{"next": {"/api/auth" + path}}.Encode()
	t.Run("without a session the user signs in first", func(t *testing.T) {
		res := r.browser(t).expect(http.MethodGet, path, nil, http.StatusFound)
		if loc := res.header.Get("Location"); loc != wantSignIn {
			t.Fatalf("sent to %s, want %s", loc, wantSignIn)
		}
	})

	t.Run("an app's token is no session", func(t *testing.T) {
		token, err := r.e.Signer.Sign(keys.Claims{Subject: user.String(), ClientID: c.ID, Scopes: []string{"user:read"}, IssuedAt: r.e.Clock.Now()})
		if err != nil {
			t.Fatal(err)
		}
		app := r.browser(t)
		app.setCookie(authn.AccessCookie, token, "/")
		if loc := app.expect(http.MethodGet, path, nil, http.StatusFound).header.Get("Location"); loc != wantSignIn {
			t.Fatalf("sent to %s, want the sign-in page", loc)
		}
	})

	t.Run("an expired session signs in again", func(t *testing.T) {
		r.e.Clock.Advance(16 * time.Minute)
		defer r.e.Clock.Advance(-16 * time.Minute)
		if loc := b.expect(http.MethodGet, path, nil, http.StatusFound).header.Get("Location"); loc != wantSignIn {
			t.Fatalf("sent to %s, want the sign-in page", loc)
		}
	})

	t.Run("the redirect can't be framed or cached", func(t *testing.T) {
		res := b.expect(http.MethodGet, path, nil, http.StatusFound)
		if res.header.Get("Cache-Control") != "no-store" || !strings.Contains(res.header.Get("Content-Security-Policy"), "frame-ancestors 'none'") {
			t.Fatalf("headers = %v", res.header)
		}
	})
}

func TestDecide(t *testing.T) {
	r := newRig(t, options{})
	user := uuid.NewV7()
	b := r.signIn(t, user)
	c, _ := createClient(t, b, "user:read")
	request := func() string {
		q := url.Values{
			"response_type": {"code"}, "client_id": {c.ID}, "redirect_uri": {appRedirect}, "state": {"st"},
			"code_challenge": {oauth2.S256ChallengeFromVerifier(oauth2.GenerateVerifier())}, "code_challenge_method": {"S256"},
		}
		return b.expect(http.MethodGet, "/oauth/authorize?"+q.Encode(), nil, http.StatusFound).location(t).Query().Get("request")
	}
	errorPage := func(reason string) string { return webURL + "/oauth/error?reason=" + reason }

	tests := []struct {
		name string
		from *browser
		form url.Values
		want string
	}{
		{"no session", r.browser(t), url.Values{"request": {request()}, "decision": {"allow"}}, errorPage("login_required")},
		{"no decision", b, url.Values{"request": {request()}}, errorPage("invalid_request")},
		{"another decision", b, url.Values{"request": {request()}, "decision": {"maybe"}}, errorPage("invalid_request")},
		{"two decisions", b, url.Values{"request": {request()}, "decision": {"allow", "deny"}}, errorPage("invalid_request")},
		{"no request", b, url.Values{"decision": {"allow"}}, errorPage("invalid_request")},
		{"a malformed request", b, url.Values{"request": {"x"}, "decision": {"allow"}}, errorPage("invalid_request")},
		{"two requests", b, url.Values{"request": {request(), request()}, "decision": {"allow"}}, errorPage("invalid_request")},
		{"an unknown request", b, url.Values{"request": {uuid.NewV4().String()}, "decision": {"allow"}}, errorPage("request_expired")},
		{"someone else's request", r.signIn(t, uuid.NewV7()), url.Values{"request": {request()}, "decision": {"allow"}}, errorPage("request_expired")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.from.form("/oauth/authorize", tt.form, nil)
			if res.status != http.StatusSeeOther || res.header.Get("Location") != tt.want {
				t.Fatalf("POST /oauth/authorize = %d to %s, want %s", res.status, res.header.Get("Location"), tt.want)
			}
		})
	}

	t.Run("deny", func(t *testing.T) {
		loc := b.form("/oauth/authorize", url.Values{"request": {request()}, "decision": {"deny"}}, nil).location(t)
		if q := loc.Query(); q.Get("error") != "access_denied" || q.Get("state") != "st" || q.Get("iss") != apptest.Issuer || q.Has("code") {
			t.Fatalf("sent to %s", loc)
		}
	})

	t.Run("a request is answered once", func(t *testing.T) {
		form := url.Values{"request": {request()}, "decision": {"allow"}}
		if loc := b.form("/oauth/authorize", form, nil).location(t); !strings.HasPrefix(loc.String(), appRedirect+"?") {
			t.Fatalf("allowed, sent to %s", loc)
		}
		if loc := b.form("/oauth/authorize", form, nil).header.Get("Location"); loc != errorPage("request_expired") {
			t.Fatalf("answered again, sent to %s", loc)
		}
	})
}

func TestTokenEndpointAuthenticatesTheClient(t *testing.T) {
	r := newRig(t, options{})
	b := r.signIn(t, uuid.NewV7())
	c, secret := createClient(t, b, "user:read")
	stored, err := r.e.Store.Client(t.Context(), c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored.SecretHash, "$argon2id$") || strings.Contains(stored.SecretHash, secret) {
		t.Fatalf("the stored secret is %q, want an argon2id hash", stored.SecretHash)
	}
	cc := url.Values{"grant_type": {"client_credentials"}}
	post := func(form url.Values, header http.Header) response {
		return r.browser(t).form("/oauth/token", form, header)
	}
	with := func(extra url.Values) url.Values {
		form := url.Values{"grant_type": {"client_credentials"}}
		maps.Copy(form, extra)
		return form
	}

	tests := []struct {
		name   string
		form   url.Values
		header http.Header
		status int
		code   string
	}{
		{"a wrong secret", cc, basic(c.ID, secret+"x"), 401, "invalid_client"},
		{"a wrong secret in the form", with(url.Values{"client_id": {c.ID}, "client_secret": {"x"}}), nil, 401, "invalid_client"},
		{"no credentials", cc, nil, 401, "invalid_client"},
		{"a bearer token", cc, http.Header{"Authorization": {"Bearer x"}}, 401, "invalid_client"},
		{"badly escaped credentials", cc, http.Header{"Authorization": {"Basic " + "JXp6OiV6eg=="}}, 401, "invalid_client"},
		{"another client_id in the form", with(url.Values{"client_id": {uuid.NewV4().String()}}), basic(c.ID, secret), 401, "invalid_client"},
		{"both ways at once", with(url.Values{"client_secret": {secret}}), basic(c.ID, secret), 400, "invalid_request"},
		{"a repeated client_id", with(url.Values{"client_id": {c.ID, c.ID}, "client_secret": {secret}}), nil, 400, "invalid_request"},
		{"no grant_type", url.Values{}, basic(c.ID, secret), 400, "invalid_request"},
		{"the password grant", url.Values{"grant_type": {"password"}}, basic(c.ID, secret), 400, "unsupported_grant_type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := post(tt.form, tt.header)
			if res.status != tt.status || res.oauthError(t) != tt.code {
				t.Fatalf("token = %d %s, want %d %s", res.status, res.body, tt.status, tt.code)
			}
			if res.header.Get("Cache-Control") != "no-store" || res.header.Get("Pragma") != "no-cache" {
				t.Errorf("cache headers = %v", res.header)
			}
			if wa := res.header.Get("WWW-Authenticate"); tt.status == 401 && !strings.HasPrefix(wa, `Basic realm=`) {
				t.Errorf("WWW-Authenticate = %q", wa)
			}
			if strings.Contains(string(res.body), secret) {
				t.Errorf("the error shows the secret: %s", res.body)
			}
		})
	}

	t.Run("a body that isn't a form", func(t *testing.T) {
		b := r.browser(t)
		res := b.send(b.request(http.MethodPost, "/oauth/token", strings.NewReader(`{"grant_type":"client_credentials"}`), http.Header{"Content-Type": {"application/json"}}))
		if res.status != 400 || res.oauthError(t) != "invalid_request" {
			t.Fatalf("token = %d %s", res.status, res.body)
		}
		res = b.send(b.request(http.MethodPost, "/oauth/token", strings.NewReader("grant_type=%zz"), http.Header{"Content-Type": {"application/x-www-form-urlencoded"}}))
		if res.status != 400 || res.oauthError(t) != "invalid_request" {
			t.Fatalf("token with a malformed form = %d %s", res.status, res.body)
		}
	})

	t.Run("parameters in the query don't count", func(t *testing.T) {
		b := r.browser(t)
		req := b.request(http.MethodPost, "/oauth/token?grant_type=client_credentials", strings.NewReader(""), basic(c.ID, secret))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if res := b.send(req); res.oauthError(t) != "invalid_request" {
			t.Fatalf("token = %s", res.body)
		}
	})

	t.Run("the right secret, either way", func(t *testing.T) {
		for _, res := range []response{
			post(cc, basic(c.ID, secret)),
			post(with(url.Values{"client_id": {c.ID}}), basic(c.ID, secret)),
			post(with(url.Values{"client_id": {c.ID}, "client_secret": {secret}}), nil),
		} {
			if res.status != http.StatusOK || res.header.Get("Cache-Control") != "no-store" || res.header.Get("Pragma") != "no-cache" {
				t.Fatalf("token = %d %v %s", res.status, res.header, res.body)
			}
		}
	})

	t.Run("revocation authenticates the client too", func(t *testing.T) {
		res := r.browser(t).form("/oauth/revoke", url.Values{"token": {"x"}}, basic(c.ID, "wrong"))
		if res.status != 401 || res.oauthError(t) != "invalid_client" {
			t.Fatalf("revoke = %d %s", res.status, res.body)
		}
		res = r.browser(t).form("/oauth/revoke", url.Values{}, basic(c.ID, secret))
		if res.status != 400 || res.oauthError(t) != "invalid_request" {
			t.Fatalf("revoke without a token = %d %s", res.status, res.body)
		}
	})
}

func TestMetadata(t *testing.T) {
	r := newRig(t, options{})
	var m map[string]any
	res := r.browser(t).expect(http.MethodGet, "/.well-known/oauth-authorization-server", nil, http.StatusOK)
	res.decode(t, &m)
	want := map[string]any{
		"issuer":                 apptest.Issuer,
		"authorization_endpoint": apptest.Issuer + "/oauth/authorize",
		"token_endpoint":         apptest.Issuer + "/oauth/token",
		"revocation_endpoint":    apptest.Issuer + "/oauth/revoke",
		"jwks_uri":               apptest.Issuer + "/.well-known/jwks.json",
		"authorization_response_iss_parameter_supported": true,
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("%s = %v, want %v", k, m[k], v)
		}
	}
	if got := m["code_challenge_methods_supported"].([]any); len(got) != 1 || got[0] != "S256" {
		t.Errorf("code_challenge_methods_supported = %v", got)
	}
	if got := m["scopes_supported"].([]any); slices.Contains(got, any(keys.ServiceScope)) || len(got) != 4 {
		t.Errorf("scopes_supported = %v", got)
	}
	if !strings.HasPrefix(res.header.Get("Cache-Control"), "public") {
		t.Errorf("Cache-Control = %q", res.header.Get("Cache-Control"))
	}
}

// keysServer serves the JWKS of k through the API.
func keysServer(t *testing.T, k *keys.Keys) *httptest.Server {
	t.Helper()
	e := apptest.New(t)
	router, err := httpapi.New(httpapi.Config{
		App: e.Service, Keys: k, Probe: run.NewProbe(nil), Verifier: e.Verifier(),
		Issuer: apptest.Issuer, WebURL: webURL, RefreshCookiePath: refreshPath, Logger: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv
}

func TestAPreviousKeyVerifiesAndARemovedOneDoesnt(t *testing.T) {
	newKey := func() *ecdsa.PrivateKey {
		k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		return k
	}
	old, current := newKey(), newKey()
	oldKeys, err := keys.New(old)
	if err != nil {
		t.Fatal(err)
	}
	token, err := keys.NewSigner(oldKeys, apptest.Issuer, apptest.Audience).Sign(keys.Claims{Subject: uuid.NewV7().String(), ClientID: "web", IssuedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	verify := func(k *keys.Keys) error {
		srv := keysServer(t, k)
		v := authn.NewVerifier(authn.VerifierConfig{JWKSURL: srv.URL + "/.well-known/jwks.json", Issuer: apptest.Issuer, Audience: apptest.Audience}, srv.Client())
		_, err := v.Verify(t.Context(), token)
		return err
	}

	rotated, err := keys.New(current, &old.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := verify(rotated); err != nil {
		t.Fatalf("a token of the previous key: %v", err)
	}
	removed, err := keys.New(current)
	if err != nil {
		t.Fatal(err)
	}
	if err := verify(removed); err == nil {
		t.Fatal("a token of a removed key verified")
	}
}

func TestNewRefusesABadConfig(t *testing.T) {
	e := apptest.New(t)
	ok := httpapi.Config{App: e.Service, Keys: e.Keys, Probe: run.NewProbe(nil), Verifier: e.Verifier(), Issuer: apptest.Issuer, WebURL: webURL, RefreshCookiePath: refreshPath, Logger: slog.New(slog.DiscardHandler)}
	for name, edit := range map[string]func(*httpapi.Config){
		"a relative issuer":      func(c *httpapi.Config) { c.Issuer = "/api/auth" },
		"a relative web URL":     func(c *httpapi.Config) { c.WebURL = "web" },
		"a relative cookie path": func(c *httpapi.Config) { c.RefreshCookiePath = "v1/sessions" },
	} {
		cfg := ok
		edit(&cfg)
		if _, err := httpapi.New(cfg); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}
