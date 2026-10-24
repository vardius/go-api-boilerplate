package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
)

func TestSignInSetsTheSessionCookies(t *testing.T) {
	r := newRig(t, options{})
	b := r.browser(t)
	r.e.Logins.Add("the-code", uuid.NewV7(), "a@example.com")

	res := b.expect(http.MethodPost, "/v1/sessions", map[string]string{"code": "the-code"}, http.StatusCreated)

	var got struct {
		Session struct {
			ID        uuid.UUID `json:"id"`
			ExpiresAt time.Time `json:"expires_at"`
		} `json:"session"`
	}
	res.decode(t, &got)
	if got.Session.ID == uuid.Nil() || !got.Session.ExpiresAt.Equal(r.e.Clock.Now().Add(30*24*time.Hour)) {
		t.Fatalf("session = %+v", got.Session)
	}
	cookies := setCookies(res)
	access, refresh := cookies["access_token"], cookies["refresh_token"]
	if access == nil || refresh == nil {
		t.Fatalf("cookies = %v", res.header["Set-Cookie"])
	}
	if !access.HttpOnly || !access.Secure || access.SameSite != http.SameSiteLaxMode || access.Path != "/" || access.MaxAge != 900 {
		t.Errorf("access cookie = %s", access)
	}
	if !refresh.HttpOnly || !refresh.Secure || refresh.SameSite != http.SameSiteStrictMode || refresh.Path != refreshPath || refresh.MaxAge != 2592000 {
		t.Errorf("refresh cookie = %s", refresh)
	}
	if id := verifyCookie(t, r, b); id != got.Session.ID.String() {
		t.Errorf("listed session %s, want %s", id, got.Session.ID)
	}
}

// verifyCookie returns the current session the browser's cookies list.
func verifyCookie(t *testing.T, r *rig, b *browser) string {
	t.Helper()
	var list struct {
		Sessions []struct {
			ID      string `json:"id"`
			Current bool   `json:"current"`
		} `json:"sessions"`
	}
	b.expect(http.MethodGet, "/v1/sessions", nil, http.StatusOK).decode(t, &list)
	for _, s := range list.Sessions {
		if s.Current {
			return s.ID
		}
	}
	return ""
}

func TestCookieSecureFalseDropsSecure(t *testing.T) {
	r := newRig(t, options{insecureCookies: true})
	b := r.browser(t)
	r.e.Logins.Add("c", uuid.NewV7(), "a@example.com")
	res := b.expect(http.MethodPost, "/v1/sessions", map[string]string{"code": "c"}, http.StatusCreated)
	for name, c := range setCookies(res) {
		if c.Secure {
			t.Errorf("%s is Secure with COOKIE_SECURE=false", name)
		}
	}
	res = b.expect(http.MethodDelete, "/v1/sessions/current", nil, http.StatusNoContent)
	for name, c := range setCookies(res) {
		if c.Secure {
			t.Errorf("clearing %s is Secure with COOKIE_SECURE=false", name)
		}
	}
}

func TestSignInRefusals(t *testing.T) {
	r := newRig(t, options{})
	b := r.browser(t)
	tests := []struct {
		name   string
		header http.Header
		body   string
		want   int
	}{
		{"an unknown code", http.Header{"Content-Type": {"application/json"}}, `{"code":"nope"}`, http.StatusUnauthorized},
		{"no code", http.Header{"Content-Type": {"application/json"}}, `{}`, http.StatusBadRequest},
		{"an unknown field", http.Header{"Content-Type": {"application/json"}}, `{"code":"c","user":"me"}`, http.StatusBadRequest},
		{"a form", http.Header{"Content-Type": {"application/x-www-form-urlencoded"}}, `code=c`, http.StatusBadRequest},
		{"a body too large", http.Header{"Content-Type": {"application/json"}}, `{"code":"` + strings.Repeat("a", 70<<10) + `"}`, http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := b.send(b.request(http.MethodPost, "/v1/sessions", strings.NewReader(tt.body), tt.header))
			if res.status != tt.want || !strings.HasPrefix(res.header.Get("Content-Type"), "application/problem+json") {
				t.Fatalf("status = %d, want %d: %s", res.status, tt.want, res.body)
			}
			if len(setCookies(res)) != 0 {
				t.Fatalf("a refused sign-in set cookies: %v", res.header["Set-Cookie"])
			}
		})
	}
}

func TestRefreshRotatesTheCookies(t *testing.T) {
	r := newRig(t, options{})
	b := r.signIn(t, uuid.NewV7())
	oldRefresh := b.cookie("refresh_token", refreshPath)

	res := b.expect(http.MethodPost, "/v1/sessions/refresh", nil, http.StatusOK)
	cookies := setCookies(res)
	if cookies["access_token"] == nil || cookies["refresh_token"] == nil || cookies["refresh_token"].Value == oldRefresh {
		t.Fatalf("refresh set %v", res.header["Set-Cookie"])
	}
	newRefresh := b.cookie("refresh_token", refreshPath)

	t.Run("a reused token ends the session and clears both cookies", func(t *testing.T) {
		thief := r.browser(t)
		thief.setCookie("refresh_token", oldRefresh, refreshPath)
		thief.setCookie("access_token", "stale", "/")
		res := thief.expect(http.MethodPost, "/v1/sessions/refresh", nil, http.StatusUnauthorized)
		cleared := setCookies(res)
		for _, name := range []string{"access_token", "refresh_token"} {
			if c := cleared[name]; c == nil || c.MaxAge >= 0 || c.Value != "" {
				t.Errorf("%s not cleared: %v", name, res.header["Set-Cookie"])
			}
		}
		if cleared["refresh_token"] != nil && cleared["refresh_token"].Path != refreshPath {
			t.Errorf("cleared the refresh cookie on path %q", cleared["refresh_token"].Path)
		}
		b.setCookie("refresh_token", newRefresh, refreshPath)
		b.expect(http.MethodPost, "/v1/sessions/refresh", nil, http.StatusUnauthorized)
	})
}

// failingRefresh is a store whose refresh token lookups fail.
type failingRefresh struct{ store.Store }

func (failingRefresh) RefreshTokenGrant(context.Context, store.Hash, time.Time) (store.Grant, bool, error) {
	return store.Grant{}, false, errors.New("database down")
}

func TestRefreshFailingOnTheServerKeepsTheCookies(t *testing.T) {
	r := newRig(t, options{store: func(s store.Store) store.Store { return failingRefresh{s} }})
	b := r.signIn(t, uuid.NewV7())
	res := b.expect(http.MethodPost, "/v1/sessions/refresh", nil, http.StatusInternalServerError)
	if len(setCookies(res)) != 0 {
		t.Fatalf("a server failure touched the cookies: %v", res.header["Set-Cookie"])
	}
	if strings.Contains(string(res.body), "database") {
		t.Fatalf("the 500 says what failed: %s", res.body)
	}
}

func TestSignOut(t *testing.T) {
	r := newRig(t, options{})
	b := r.signIn(t, uuid.NewV7())
	refresh := b.cookie("refresh_token", refreshPath)
	res := b.expect(http.MethodDelete, "/v1/sessions/current", nil, http.StatusNoContent)
	if c := setCookies(res); c["access_token"] == nil || c["refresh_token"] == nil {
		t.Fatalf("sign out set %v", res.header["Set-Cookie"])
	}
	b.setCookie("refresh_token", refresh, refreshPath)
	b.expect(http.MethodPost, "/v1/sessions/refresh", nil, http.StatusUnauthorized)
}

func TestRevokeAnotherSession(t *testing.T) {
	r := newRig(t, options{})
	user := uuid.NewV7()
	laptop, phone := r.signIn(t, user), r.signIn(t, user)
	phoneSession := verifyCookie(t, r, phone)

	laptop.expect(http.MethodDelete, "/v1/sessions/"+phoneSession, nil, http.StatusNoContent)
	phone.expect(http.MethodPost, "/v1/sessions/refresh", nil, http.StatusUnauthorized)

	for name, id := range map[string]string{"gone": phoneSession, "malformed": "current-ish", "unknown": uuid.NewV7().String()} {
		t.Run(name, func(t *testing.T) {
			laptop.expect(http.MethodDelete, "/v1/sessions/"+id, nil, http.StatusNotFound)
		})
	}
	t.Run("someone else's", func(t *testing.T) {
		r.signIn(t, uuid.NewV7()).expect(http.MethodDelete, "/v1/sessions/"+verifyCookie(t, r, laptop), nil, http.StatusNotFound)
	})
}

func TestSignInIsRateLimited(t *testing.T) {
	r := newRig(t, options{rateLimit: httpx.RateLimitConfig{Rate: 0.001, Burst: 1}})
	b := r.browser(t)
	b.expect(http.MethodPost, "/v1/sessions", map[string]string{"code": "a"}, http.StatusUnauthorized)
	res := b.expect(http.MethodPost, "/v1/sessions", map[string]string{"code": "b"}, http.StatusTooManyRequests)
	if res.header.Get("Retry-After") == "" {
		t.Fatal("no Retry-After")
	}
	// The token endpoint has its own budget.
	b.form("/oauth/token", nil, nil)
	if res := b.form("/oauth/token", nil, nil); res.status != http.StatusTooManyRequests {
		t.Fatalf("second token request = %d, want 429", res.status)
	}
	// Other routes aren't limited.
	b.expect(http.MethodGet, "/.well-known/jwks.json", nil, http.StatusOK)
}

func TestCookieRoutesRefuseOtherOrigins(t *testing.T) {
	r := newRig(t, options{})
	b := r.signIn(t, uuid.NewV7())
	cross := http.Header{"Sec-Fetch-Site": {"cross-site"}, "Origin": {"https://evil.example"}}
	for _, route := range []struct{ method, path string }{
		{http.MethodPost, "/v1/sessions"},
		{http.MethodPost, "/v1/sessions/refresh"},
		{http.MethodDelete, "/v1/sessions/current"},
		{http.MethodDelete, "/v1/sessions/" + uuid.NewV7().String()},
		{http.MethodPost, "/v1/clients"},
		{http.MethodDelete, "/v1/clients/x"},
		{http.MethodPost, "/oauth/authorize"},
	} {
		if res := b.do(route.method, route.path, nil, cross); res.status != http.StatusForbidden {
			t.Errorf("cross-origin %s %s = %d, want 403", route.method, route.path, res.status)
		}
	}
	// The session survived all of that.
	b.expect(http.MethodPost, "/v1/sessions/refresh", nil, http.StatusOK)

	// Clients call the token endpoint from anywhere.
	if res := b.form("/oauth/token", nil, cross); res.status != http.StatusUnauthorized || res.oauthError(t) != "invalid_client" {
		t.Errorf("cross-origin token request = %d %s, want invalid_client (not 403)", res.status, res.body)
	}
}
