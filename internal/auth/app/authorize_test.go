package app_test

import (
	"errors"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"golang.org/x/oauth2"

	"github.com/vardius/go-api-boilerplate/internal/auth/app"
	"github.com/vardius/go-api-boilerplate/internal/auth/app/apptest"
	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
)

func authorizeQuery(c store.Client, challenge string) url.Values {
	return url.Values{
		"response_type":         {"code"},
		"client_id":             {c.ID},
		"redirect_uri":          {redirectURI},
		"state":                 {"the state"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
}

// authorize takes q through consent to the code it is answered with.
func authorize(t *testing.T, e *apptest.Env, user uuid.UUID, q url.Values) string {
	t.Helper()
	a := must(e.Service.CheckAuthorization(t.Context(), q))(t)
	id := must(e.Service.RequestConsent(t.Context(), user, a))(t)
	back := must(url.Parse(must(e.Service.Decide(t.Context(), user, id, true))(t)))(t)
	if back.Query().Get("state") != q.Get("state") || back.Query().Get("iss") != apptest.Issuer {
		t.Fatalf("redirected to %s", back)
	}
	return back.Query().Get("code")
}

func exchange(code, verifier string) url.Values {
	return url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {redirectURI}, "code_verifier": {verifier}}
}

// authorizationCodeTokens returns the tokens c gets for user through the
// whole authorization code flow.
func authorizationCodeTokens(t *testing.T, e *apptest.Env, user uuid.UUID, c store.Client) app.Tokens {
	t.Helper()
	verifier := oauth2.GenerateVerifier()
	code := authorize(t, e, user, authorizeQuery(c, oauth2.S256ChallengeFromVerifier(verifier)))
	return must(e.Service.Token(t.Context(), c, exchange(code, verifier)))(t)
}

func TestCheckAuthorization(t *testing.T) {
	e := apptest.New(t)
	c, _ := newClient(t, e, uuid.NewV7(), "user:read", "clients:read")
	challenge := oauth2.S256ChallengeFromVerifier(oauth2.GenerateVerifier())

	a := must(e.Service.CheckAuthorization(t.Context(), authorizeQuery(c, challenge)))(t)
	if a.Client.ID != c.ID || a.RedirectURI != redirectURI || a.State != "the state" || a.CodeChallenge != challenge ||
		strings.Join(a.Scopes, " ") != "user:read clients:read" {
		t.Fatalf("authorization = %+v", a)
	}
	q := authorizeQuery(c, challenge)
	q.Set("scope", "clients:read")
	q.Del("state")
	if a := must(e.Service.CheckAuthorization(t.Context(), q))(t); strings.Join(a.Scopes, " ") != "clients:read" || a.State != "" {
		t.Fatalf("authorization for one scope = %+v", a)
	}
}

// No failed authorization request goes back to the client, not even to a
// redirect URI it registered: any user can register a client, so that would
// be an open redirector (RFC 9700 §4.11.2). Each fails with a reason for the
// web app's error page.
func TestCheckAuthorizationNeverRedirects(t *testing.T) {
	e := apptest.New(t)
	c, _ := newClient(t, e, uuid.NewV7(), "user:read")
	challenge := oauth2.S256ChallengeFromVerifier(oauth2.GenerateVerifier())
	tests := []struct {
		name   string
		edit   func(url.Values)
		reason string
	}{
		{"no client_id", func(q url.Values) { q.Del("client_id") }, app.ReasonInvalidClient},
		{"two client_ids", func(q url.Values) { q.Add("client_id", c.ID) }, app.ReasonInvalidClient},
		{"a malformed client_id", func(q url.Values) { q.Set("client_id", "web") }, app.ReasonInvalidClient},
		{"an unknown client", func(q url.Values) { q.Set("client_id", uuid.NewV4().String()) }, app.ReasonInvalidClient},
		{"no redirect_uri", func(q url.Values) { q.Del("redirect_uri") }, app.ReasonInvalidRedirectURI},
		{"two redirect_uris", func(q url.Values) { q.Add("redirect_uri", redirectURI) }, app.ReasonInvalidRedirectURI},
		{"a longer path", func(q url.Values) { q.Set("redirect_uri", redirectURI+"/more") }, app.ReasonInvalidRedirectURI},
		{"an added query", func(q url.Values) { q.Set("redirect_uri", redirectURI+"?x=1") }, app.ReasonInvalidRedirectURI},
		{"another case", func(q url.Values) { q.Set("redirect_uri", strings.ToUpper(redirectURI)) }, app.ReasonInvalidRedirectURI},
		{"a trailing slash", func(q url.Values) { q.Set("redirect_uri", redirectURI+"/") }, app.ReasonInvalidRedirectURI},
		{"another port on localhost", func(q url.Values) { q.Set("redirect_uri", "http://localhost:9998/cb") }, app.ReasonInvalidRedirectURI},
		{"a repeated parameter", func(q url.Values) { q.Add("response_type", "code") }, app.ReasonInvalidRequest},
		{"a repeated state", func(q url.Values) { q.Add("state", "other") }, app.ReasonInvalidRequest},
		{"a long state", func(q url.Values) { q.Set("state", strings.Repeat("s", 1025)) }, app.ReasonInvalidRequest},
		{"a state with a control character", func(q url.Values) { q.Set("state", "a\nb") }, app.ReasonInvalidRequest},
		{"no response_type", func(q url.Values) { q.Del("response_type") }, app.ReasonInvalidRequest},
		{"the implicit grant", func(q url.Values) { q.Set("response_type", "token") }, app.ReasonUnsupportedResponseType},
		{"no PKCE", func(q url.Values) { q.Del("code_challenge"); q.Del("code_challenge_method") }, app.ReasonPKCERequired},
		{"PKCE plain", func(q url.Values) { q.Set("code_challenge_method", "plain") }, app.ReasonPKCERequired},
		{"no method, which means plain", func(q url.Values) { q.Del("code_challenge_method") }, app.ReasonPKCERequired},
		{"a short challenge", func(q url.Values) { q.Set("code_challenge", challenge[1:]) }, app.ReasonPKCERequired},
		{"a challenge outside base64url", func(q url.Values) { q.Set("code_challenge", challenge[1:]+"=") }, app.ReasonPKCERequired},
		{"a scope wider than the client's", func(q url.Values) { q.Set("scope", "user:read user:write") }, app.ReasonInvalidScope},
		{"an unknown scope", func(q url.Values) { q.Set("scope", "everything") }, app.ReasonInvalidScope},
		{"the service scope", func(q url.Values) { q.Set("scope", "login-codes:redeem") }, app.ReasonInvalidScope},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := authorizeQuery(c, challenge)
			tt.edit(q)
			_, err := e.Service.CheckAuthorization(t.Context(), q)
			ae, ok := errors.AsType[*app.AuthorizeError](err)
			if !ok || ae.Code != tt.reason || ae.Description == "" {
				t.Fatalf("err = %#v, want %s", err, tt.reason)
			}
			if !slices.Contains(app.Reasons(), ae.Code) {
				t.Fatalf("%s isn't one of the Reasons", ae.Code)
			}
		})
	}
}

func TestRedirectKeepsTheRegisteredQuery(t *testing.T) {
	e := apptest.New(t)
	const withQuery = "https://app.example.test/cb?tenant=7"
	c, _, err := e.Service.CreateClient(t.Context(), uuid.NewV7(), app.NewClient{Name: "x", RedirectURIs: []string{withQuery}, Scopes: []string{"user:read"}})
	if err != nil {
		t.Fatal(err)
	}
	verifier := oauth2.GenerateVerifier()
	q := authorizeQuery(c, oauth2.S256ChallengeFromVerifier(verifier))
	q.Set("redirect_uri", withQuery)
	user := uuid.NewV7()
	id := must(e.Service.RequestConsent(t.Context(), user, must(e.Service.CheckAuthorization(t.Context(), q))(t)))(t)
	back := must(url.Parse(must(e.Service.Decide(t.Context(), user, id, true))(t)))(t)
	if back.Query().Get("tenant") != "7" || back.Query().Get("code") == "" {
		t.Fatalf("redirected to %s", back)
	}
}

func TestConsent(t *testing.T) {
	e := apptest.New(t)
	user := uuid.NewV7()
	c, _ := newClient(t, e, uuid.NewV7(), "user:read", "clients:write")
	a := must(e.Service.CheckAuthorization(t.Context(), authorizeQuery(c, oauth2.S256ChallengeFromVerifier(oauth2.GenerateVerifier()))))(t)
	id := must(e.Service.RequestConsent(t.Context(), user, a))(t)

	got := must(e.Service.Consent(t.Context(), user, id))(t)
	if got.ClientID != c.ID || got.ClientName != "The App" || got.RedirectURI != redirectURI || len(got.Scopes) != 2 ||
		got.Scopes[0].Name != "user:read" || got.Scopes[1].Name != "clients:write" || got.Scopes[0].Description == "" {
		t.Fatalf("consent = %+v", got)
	}

	t.Run("someone else's request is not found", func(t *testing.T) {
		_, err := e.Service.Consent(t.Context(), uuid.NewV7(), id)
		isKind(t, err, apperr.ErrNotFound)
		_, err = e.Service.Decide(t.Context(), uuid.NewV7(), id, true)
		if ae, ok := errors.AsType[*app.AuthorizeError](err); !ok || ae.Code != app.ReasonRequestExpired {
			t.Fatalf("deciding someone else's request: %v", err)
		}
		must(e.Service.Consent(t.Context(), user, id))(t)
	})

	t.Run("a request expires", func(t *testing.T) {
		e.Clock.Advance(app.AuthorizationRequestTTL)
		_, err := e.Service.Consent(t.Context(), user, id)
		isKind(t, err, apperr.ErrNotFound)
	})

	t.Run("a request of a deleted client is not found", func(t *testing.T) {
		id := must(e.Service.RequestConsent(t.Context(), user, a))(t)
		if err := e.Service.DeleteClient(t.Context(), c.OwnerID, c.ID); err != nil {
			t.Fatal(err)
		}
		_, err := e.Service.Consent(t.Context(), user, id)
		isKind(t, err, apperr.ErrNotFound)
	})
}

func TestDecide(t *testing.T) {
	e := apptest.New(t)
	user := uuid.NewV7()
	c, _ := newClient(t, e, uuid.NewV7(), "user:read")
	a := must(e.Service.CheckAuthorization(t.Context(), authorizeQuery(c, oauth2.S256ChallengeFromVerifier(oauth2.GenerateVerifier()))))(t)

	t.Run("deny", func(t *testing.T) {
		id := must(e.Service.RequestConsent(t.Context(), user, a))(t)
		back := must(url.Parse(must(e.Service.Decide(t.Context(), user, id, false))(t)))(t)
		q := back.Query()
		if q.Get("error") != app.ErrorAccessDenied || q.Get("state") != "the state" || q.Get("iss") != apptest.Issuer || q.Has("code") {
			t.Fatalf("redirected to %s", back)
		}
		_, err := e.Service.Decide(t.Context(), user, id, true)
		if ae, ok := errors.AsType[*app.AuthorizeError](err); !ok || ae.Code != app.ReasonRequestExpired {
			t.Fatalf("deciding twice: %v", err)
		}
	})

	t.Run("allow issues a one-minute code", func(t *testing.T) {
		id := must(e.Service.RequestConsent(t.Context(), user, a))(t)
		back := must(url.Parse(must(e.Service.Decide(t.Context(), user, id, true))(t)))(t)
		code := must(e.Store.AuthorizationCode(t.Context(), store.HashToken(back.Query().Get("code"))))(t)
		if code.ClientID != c.ID || code.UserID != user || code.RedirectURI != redirectURI || code.CodeChallenge != a.CodeChallenge ||
			code.Used || !code.ExpiresAt.Equal(e.Clock.Now().Add(time.Minute)) {
			t.Fatalf("code = %+v", code)
		}
	})

	t.Run("a client deleted meanwhile takes its requests", func(t *testing.T) {
		id := must(e.Service.RequestConsent(t.Context(), user, a))(t)
		if err := e.Service.DeleteClient(t.Context(), c.OwnerID, c.ID); err != nil {
			t.Fatal(err)
		}
		_, err := e.Service.Decide(t.Context(), user, id, true)
		if ae, ok := errors.AsType[*app.AuthorizeError](err); !ok || ae.Code != app.ReasonRequestExpired {
			t.Fatalf("err = %v, want %s", err, app.ReasonRequestExpired)
		}
	})
}
