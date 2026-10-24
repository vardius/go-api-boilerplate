package app_test

import (
	"errors"
	"net/url"
	"strings"
	"sync"
	"testing"
	"uuid"

	"golang.org/x/oauth2"

	"github.com/vardius/go-api-boilerplate/internal/auth/app"
	"github.com/vardius/go-api-boilerplate/internal/auth/app/apptest"
	"github.com/vardius/go-api-boilerplate/internal/auth/keys"
	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
)

func TestTokenRequestRefusals(t *testing.T) {
	e := apptest.New(t)
	c, _ := newClient(t, e, uuid.NewV7(), "user:read")
	tests := []struct {
		name string
		form url.Values
		code string
	}{
		{"no grant_type", url.Values{}, app.ErrorInvalidRequest},
		{"an empty grant_type", url.Values{"grant_type": {""}}, app.ErrorInvalidRequest},
		{"a repeated grant_type", url.Values{"grant_type": {"client_credentials", "client_credentials"}}, app.ErrorInvalidRequest},
		{"the password grant", url.Values{"grant_type": {"password"}}, app.ErrorUnsupportedGrantType},
		{"no code", url.Values{"grant_type": {"authorization_code"}}, app.ErrorInvalidRequest},
		{"an unknown code", exchange("unknown", oauth2.GenerateVerifier()), app.ErrorInvalidGrant},
		{"no refresh_token", url.Values{"grant_type": {"refresh_token"}}, app.ErrorInvalidRequest},
		{"an unknown refresh_token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"unknown"}}, app.ErrorInvalidGrant},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := e.Service.Token(t.Context(), c, tt.form)
			if oe := oauthCode(t, err, tt.code); oe.Description == "" {
				t.Fatal("no error_description")
			}
		})
	}
}

func TestAuthorizationCodeGrant(t *testing.T) {
	e := apptest.New(t)
	user := uuid.NewV7()
	c, _ := newClient(t, e, uuid.NewV7(), "user:read", "clients:read")
	verifier := oauth2.GenerateVerifier()
	q := authorizeQuery(c, oauth2.S256ChallengeFromVerifier(verifier))
	q.Set("scope", "user:read")
	code := authorize(t, e, user, q)

	tokens := must(e.Service.Token(t.Context(), c, exchange(code, verifier)))(t)

	id := verify(t, e, tokens.AccessToken.Reveal())
	if id.UserID != user || id.ClientID != c.ID || id.IsFirstParty() || strings.Join(id.Scopes, " ") != "user:read" {
		t.Fatalf("access token for %+v", id)
	}
	if tokens.RefreshToken == "" || tokens.ExpiresIn != keys.AccessTokenTTL || strings.Join(tokens.Scopes, " ") != "user:read" {
		t.Fatalf("tokens = %+v", tokens)
	}
	g := must(e.Store.Grant(t.Context(), tokens.Grant.ID, e.Clock.Now()))(t)
	if g.UserID != user || g.ClientID != c.ID || strings.Join(g.Scopes, " ") != "user:read" || g.UserAgent != "" {
		t.Fatalf("grant = %+v", g)
	}

	t.Run("a code used twice fails and revokes what it issued", func(t *testing.T) {
		_, err := e.Service.Token(t.Context(), c, exchange(code, verifier))
		oauthCode(t, err, app.ErrorInvalidGrant)
		noSecret(t, err, code, verifier)
		_, err = e.Store.Grant(t.Context(), tokens.Grant.ID, e.Clock.Now())
		isKind(t, err, apperr.ErrNotFound)
		_, err = e.Service.Token(t.Context(), c, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tokens.RefreshToken.Reveal()}})
		oauthCode(t, err, app.ErrorInvalidGrant)
	})
}

func TestAuthorizationCodeMismatchesSpendTheCode(t *testing.T) {
	e := apptest.New(t)
	user := uuid.NewV7()
	c, _ := newClient(t, e, uuid.NewV7(), "user:read")
	other, _ := newClient(t, e, uuid.NewV7(), "user:read")
	verifier := oauth2.GenerateVerifier()

	tests := []struct {
		name   string
		client store.Client
		edit   func(url.Values)
	}{
		{"no verifier", c, func(f url.Values) { f.Del("code_verifier") }},
		{"a wrong verifier", c, func(f url.Values) { f.Set("code_verifier", oauth2.GenerateVerifier()) }},
		{"the challenge as the verifier, as plain would have it", c, func(f url.Values) { f.Set("code_verifier", oauth2.S256ChallengeFromVerifier(verifier)) }},
		{"a short verifier", c, func(f url.Values) { f.Set("code_verifier", verifier[:42]) }},
		{"a long verifier", c, func(f url.Values) { f.Set("code_verifier", strings.Repeat("a", 129)) }},
		{"a verifier outside the unreserved characters", c, func(f url.Values) { f.Set("code_verifier", verifier[1:]+"+") }},
		{"no redirect_uri", c, func(f url.Values) { f.Del("redirect_uri") }},
		{"another redirect_uri", c, func(f url.Values) { f.Set("redirect_uri", "http://localhost:9999/cb") }},
		{"another client", other, func(url.Values) {}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := authorize(t, e, user, authorizeQuery(c, oauth2.S256ChallengeFromVerifier(verifier)))
			form := exchange(code, verifier)
			tt.edit(form)
			_, err := e.Service.Token(t.Context(), tt.client, form)
			oauthCode(t, err, app.ErrorInvalidGrant)

			_, err = e.Service.Token(t.Context(), c, exchange(code, verifier))
			oauthCode(t, err, app.ErrorInvalidGrant)
		})
	}

	t.Run("an expired code", func(t *testing.T) {
		code := authorize(t, e, user, authorizeQuery(c, oauth2.S256ChallengeFromVerifier(verifier)))
		e.Clock.Advance(app.AuthorizationCodeTTL)
		_, err := e.Service.Token(t.Context(), c, exchange(code, verifier))
		oauthCode(t, err, app.ErrorInvalidGrant)
	})
}

func TestConcurrentExchangesOfOneCode(t *testing.T) {
	e := apptest.New(t)
	c, _ := newClient(t, e, uuid.NewV7(), "user:read")
	verifier := oauth2.GenerateVerifier()
	code := authorize(t, e, uuid.NewV7(), authorizeQuery(c, oauth2.S256ChallengeFromVerifier(verifier)))

	var wg sync.WaitGroup
	results := make([]error, 8)
	issued := make([]app.Tokens, len(results))
	for i := range results {
		wg.Go(func() { issued[i], results[i] = e.Service.Token(t.Context(), c, exchange(code, verifier)) })
	}
	wg.Wait()

	won := -1
	for i, err := range results {
		switch {
		case err == nil && won >= 0:
			t.Fatal("two exchanges of one code succeeded")
		case err == nil:
			won = i
		default:
			oauthCode(t, err, app.ErrorInvalidGrant)
		}
	}
	if won < 0 {
		t.Fatal("no exchange succeeded")
	}
	if _, err := e.Store.Grant(t.Context(), issued[won].Grant.ID, e.Clock.Now()); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("the grant survived its code being replayed: %v", err)
	}
}

func TestClientCredentialsGrant(t *testing.T) {
	e := apptest.New(t)
	c, _ := newClient(t, e, uuid.NewV7(), "user:read", "clients:read")

	for scope, want := range map[string]string{"": "user:read clients:read", "clients:read": "clients:read", "clients:read user:read": "user:read clients:read"} {
		tokens := must(e.Service.Token(t.Context(), c, url.Values{"grant_type": {"client_credentials"}, "scope": {scope}}))(t)
		id := verify(t, e, tokens.AccessToken.Reveal())
		if id.Subject != c.ID || id.ClientID != c.ID || id.IsUser() || strings.Join(id.Scopes, " ") != want {
			t.Errorf("scope %q: token for %+v", scope, id)
		}
		if tokens.RefreshToken != "" || tokens.Grant.ID != uuid.Nil() {
			t.Errorf("scope %q: a refresh token or grant for a client's own token: %+v", scope, tokens)
		}
	}

	for _, scope := range []string{"user:write", "user:read user:write", "login-codes:redeem", "*"} {
		_, err := e.Service.Token(t.Context(), c, url.Values{"grant_type": {"client_credentials"}, "scope": {scope}})
		oauthCode(t, err, app.ErrorInvalidScope)
	}
}

func refreshForm(token, scope string) url.Values {
	return url.Values{"grant_type": {"refresh_token"}, "refresh_token": {token}, "scope": {scope}}
}

func TestRefreshTokenGrant(t *testing.T) {
	e := apptest.New(t)
	user := uuid.NewV7()
	c, _ := newClient(t, e, uuid.NewV7(), "user:read", "clients:read")
	first := authorizationCodeTokens(t, e, user, c)

	t.Run("a scope wider than the grant's leaves the token valid", func(t *testing.T) {
		_, err := e.Service.Token(t.Context(), c, refreshForm(first.RefreshToken.Reveal(), "user:write"))
		oauthCode(t, err, app.ErrorInvalidScope)
	})

	t.Run("another client's token leaves the grant", func(t *testing.T) {
		other, _ := newClient(t, e, uuid.NewV7(), "user:read")
		_, err := e.Service.Token(t.Context(), other, refreshForm(first.RefreshToken.Reveal(), ""))
		oauthCode(t, err, app.ErrorInvalidGrant)
		must(e.Store.Grant(t.Context(), first.Grant.ID, e.Clock.Now()))(t)
	})

	second := must(e.Service.Token(t.Context(), c, refreshForm(first.RefreshToken.Reveal(), "clients:read")))(t)
	if second.RefreshToken == first.RefreshToken || second.Grant.ID != first.Grant.ID || strings.Join(second.Scopes, " ") != "clients:read" {
		t.Fatalf("refresh gave %+v", second)
	}
	if id := verify(t, e, second.AccessToken.Reveal()); id.UserID != user || strings.Join(id.Scopes, " ") != "clients:read" {
		t.Fatalf("refreshed token for %+v", id)
	}
	third := must(e.Service.Token(t.Context(), c, refreshForm(second.RefreshToken.Reveal(), "")))(t)
	if strings.Join(third.Scopes, " ") != "user:read clients:read" {
		t.Fatalf("a narrowed refresh narrowed the grant: %v", third.Scopes)
	}

	t.Run("reuse revokes the grant", func(t *testing.T) {
		_, err := e.Service.Token(t.Context(), c, refreshForm(first.RefreshToken.Reveal(), ""))
		oauthCode(t, err, app.ErrorInvalidGrant)
		noSecret(t, err, first.RefreshToken.Reveal())
		_, err = e.Service.Token(t.Context(), c, refreshForm(third.RefreshToken.Reveal(), ""))
		oauthCode(t, err, app.ErrorInvalidGrant)
		_, err = e.Store.Grant(t.Context(), first.Grant.ID, e.Clock.Now())
		isKind(t, err, apperr.ErrNotFound)
	})
}

func TestConcurrentRefreshesOfOneToken(t *testing.T) {
	e := apptest.New(t)
	c, _ := newClient(t, e, uuid.NewV7(), "user:read")
	first := authorizationCodeTokens(t, e, uuid.NewV7(), c)

	var wg sync.WaitGroup
	results := make([]error, 8)
	for i := range results {
		wg.Go(func() { _, results[i] = e.Service.Token(t.Context(), c, refreshForm(first.RefreshToken.Reveal(), "")) })
	}
	wg.Wait()
	ok := 0
	for _, err := range results {
		if err == nil {
			ok++
		} else {
			oauthCode(t, err, app.ErrorInvalidGrant)
		}
	}
	if ok > 1 {
		t.Fatalf("%d refreshes of one token succeeded", ok)
	}
	if _, err := e.Store.Grant(t.Context(), first.Grant.ID, e.Clock.Now()); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("the grant survived concurrent reuse: %v", err)
	}
}

func TestRevoke(t *testing.T) {
	e := apptest.New(t)
	c, _ := newClient(t, e, uuid.NewV7(), "user:read")
	other, _ := newClient(t, e, uuid.NewV7(), "user:read")
	tokens := authorizationCodeTokens(t, e, uuid.NewV7(), c)

	for name, form := range map[string]url.Values{
		"no token":          {},
		"a repeated token":  {"token": {"a", "b"}},
		"a repeated hint":   {"token": {"a"}, "token_type_hint": {"refresh_token", "access_token"}},
		"an empty token":    {"token": {""}},
		"an empty and more": {"token": {"", ""}},
	} {
		t.Run(name, func(t *testing.T) {
			oauthCode(t, e.Service.Revoke(t.Context(), c, form), app.ErrorInvalidRequest)
		})
	}

	ignored := []struct {
		name   string
		client store.Client
		form   url.Values
	}{
		{"an unknown token", c, url.Values{"token": {"unknown"}}},
		{"an access token", c, url.Values{"token": {tokens.AccessToken.Reveal()}, "token_type_hint": {"access_token"}}},
		{"an unknown hint", c, url.Values{"token": {"unknown"}, "token_type_hint": {"id_token"}}},
		{"another client's refresh token", other, url.Values{"token": {tokens.RefreshToken.Reveal()}}},
	}
	for _, tt := range ignored {
		if err := e.Service.Revoke(t.Context(), tt.client, tt.form); err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
	}
	must(e.Store.Grant(t.Context(), tokens.Grant.ID, e.Clock.Now()))(t)

	if err := e.Service.Revoke(t.Context(), c, url.Values{"token": {tokens.RefreshToken.Reveal()}, "token_type_hint": {"refresh_token"}}); err != nil {
		t.Fatal(err)
	}
	_, err := e.Store.Grant(t.Context(), tokens.Grant.ID, e.Clock.Now())
	isKind(t, err, apperr.ErrNotFound)
	if err := e.Service.Revoke(t.Context(), c, url.Values{"token": {tokens.RefreshToken.Reveal()}}); err != nil {
		t.Fatalf("revoking again: %v", err)
	}
}

func TestGrantableScopes(t *testing.T) {
	got := app.GrantableScopes()
	if strings.Join(got, " ") != "user:read user:write clients:read clients:write" {
		t.Fatalf("grantable = %v", got)
	}
	got[0] = "changed"
	if app.GrantableScopes()[0] != "user:read" {
		t.Fatal("GrantableScopes shares its slice")
	}
}
