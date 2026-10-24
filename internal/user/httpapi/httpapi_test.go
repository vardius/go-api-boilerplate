package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
)

func TestProbes(t *testing.T) {
	r := newRig(t, options{})
	if res := r.expect(r.browser(), http.MethodGet, "/livez", nil, http.StatusOK); string(res.body) != "ok\n" {
		t.Errorf("livez = %q", res.body)
	}
	r.expect(r.browser(), http.MethodGet, "/readyz", nil, http.StatusOK)
}

func TestConfig(t *testing.T) {
	tests := []struct {
		name string
		opts options
		want string
	}{
		{"defaults", options{}, `{"google_enabled":false}`},
		{"with Google and a mail preview", options{google: true, preview: "http://localhost:8025"}, `{"google_enabled":true,"mail_preview_url":"http://localhost:8025"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRig(t, tt.opts)
			res := r.expect(r.browser(), http.MethodGet, "/v1/config", nil, http.StatusOK)
			if got := strings.TrimSpace(string(res.body)); got != tt.want {
				t.Errorf("config = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestRequestLoginLink(t *testing.T) {
	r := newRig(t, options{})
	res := r.expect(r.browser(), http.MethodPost, "/v1/login-links", map[string]string{"email": "Ada@Example.com", "next": "/consent?request=42"}, http.StatusAccepted)
	if got := strings.TrimSpace(string(res.body)); got != "{}" {
		t.Errorf("body = %s, want {}", got)
	}
	links := r.mailed()
	if len(links) != 1 || links[0].To != "ada@example.com" || !strings.HasPrefix(links[0].URL, webURL+"/sign-in/complete?code=") ||
		!strings.HasSuffix(links[0].URL, "&next=%2Fconsent%3Frequest%3D42") {
		t.Errorf("mailed %+v", links)
	}
}

func TestRequestLoginLinkRejectsBadRequests(t *testing.T) {
	r := newRig(t, options{})
	tests := []struct {
		name   string
		body   any
		status int
		field  string
	}{
		{"a bad address", map[string]string{"email": "ada"}, http.StatusBadRequest, "email"},
		{"no address", map[string]string{}, http.StatusBadRequest, "email"},
		{"an unknown field", map[string]string{"email": "ada@example.com", "admin": "true"}, http.StatusBadRequest, "admin"},
		{"not JSON", "email=ada@example.com", http.StatusBadRequest, ""},
		{"a huge body", map[string]string{"email": strings.Repeat("a", 5000) + "@example.com"}, http.StatusRequestEntityTooLarge, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := r.expect(r.browser(), http.MethodPost, "/v1/login-links", tt.body, tt.status)
			if p := res.problem(t); tt.field != "" && p.Errors[tt.field] == "" {
				t.Errorf("problem = %+v, want an error for %s", p, tt.field)
			}
		})
	}
	if links := r.mailed(); len(links) != 0 {
		t.Errorf("mailed %+v", links)
	}
}

func TestRequestLoginLinkNeedsJSON(t *testing.T) {
	r := newRig(t, options{})
	res := r.do(r.browser(), http.MethodPost, "/v1/login-links", `{"email":"ada@example.com"}`, header{"Content-Type": "text/plain"})
	if res.status != http.StatusBadRequest {
		t.Fatalf("status = %d: %s", res.status, res.body)
	}
}

func TestRequestLoginLinkIsLimitedPerAddress(t *testing.T) {
	r := newRig(t, options{rateLimit: httpx.RateLimitConfig{Rate: 100, Burst: 100}})
	for i := range 5 {
		email := "ada@example.com"
		if i%2 == 1 {
			email = "ADA@example.com "
		}
		r.expect(r.browser(), http.MethodPost, "/v1/login-links", map[string]string{"email": email}, http.StatusAccepted)
	}
	res := r.expect(r.browser(), http.MethodPost, "/v1/login-links", map[string]string{"email": "Ada@Example.com"}, http.StatusTooManyRequests)
	if got := res.header.Get("Retry-After"); got != "60" {
		t.Errorf("Retry-After = %q, want 60", got)
	}
	// Other addresses aren't held up by it.
	r.expect(r.browser(), http.MethodPost, "/v1/login-links", map[string]string{"email": "grace@example.com"}, http.StatusAccepted)

	r.advance(time.Minute)
	r.expect(r.browser(), http.MethodPost, "/v1/login-links", map[string]string{"email": "ada@example.com"}, http.StatusAccepted)
	if n := len(r.mailed()); n != 7 {
		t.Errorf("mailed %d links, want 7", n)
	}
}

func TestRequestLoginLinkIsLimitedPerClient(t *testing.T) {
	r := newRig(t, options{rateLimit: httpx.RateLimitConfig{Rate: 0.001, Burst: 2}})
	for _, email := range []string{"a@example.com", "b@example.com"} {
		r.expect(r.browser(), http.MethodPost, "/v1/login-links", map[string]string{"email": email}, http.StatusAccepted)
	}
	r.expect(r.browser(), http.MethodPost, "/v1/login-links", map[string]string{"email": "c@example.com"}, http.StatusTooManyRequests)
	// Only login links are limited.
	r.expect(r.browser(), http.MethodGet, "/v1/config", nil, http.StatusOK)
}

func TestMe(t *testing.T) {
	r := newRig(t, options{})
	u := r.signUp("ada@example.com")
	var got struct {
		ID           string    `json:"id"`
		Email        string    `json:"email"`
		GoogleLinked bool      `json:"google_linked"`
		CreatedAt    time.Time `json:"created_at"`
	}
	r.expect(r.as(r.ownerToken(u)), http.MethodGet, "/v1/me", nil, http.StatusOK).decode(t, &got)
	if got.ID != u.ID.String() || got.Email != "ada@example.com" || got.GoogleLinked || !got.CreatedAt.Equal(t0) {
		t.Errorf("me = %+v", got)
	}
}

func TestMeWithABadToken(t *testing.T) {
	r := newRig(t, options{})
	res := r.expect(r.as("not-a-jwt"), http.MethodGet, "/v1/me", nil, http.StatusUnauthorized)
	if res.header.Get("WWW-Authenticate") != "Bearer" {
		t.Errorf("WWW-Authenticate = %q", res.header.Get("WWW-Authenticate"))
	}
	// A stale token in a cookie doesn't stop routes that need no one.
	r.expect(r.as("not-a-jwt"), http.MethodGet, "/v1/config", nil, http.StatusOK)
	res = r.do(r.browser(), http.MethodPost, "/v1/login-links", map[string]string{"email": "ada@example.com"}, header{"Cookie": "access_token=not-a-jwt"})
	if res.status != http.StatusAccepted {
		t.Errorf("login link with a stale cookie = %d: %s", res.status, res.body)
	}
}

func TestMeFromTheSessionCookie(t *testing.T) {
	r := newRig(t, options{})
	u := r.signUp("ada@example.com")
	res := r.do(r.browser(), http.MethodGet, "/v1/me", nil, header{"Cookie": "access_token=" + r.ownerToken(u)})
	if res.status != http.StatusOK {
		t.Fatalf("status = %d: %s", res.status, res.body)
	}
}

func TestChangeEmail(t *testing.T) {
	r := newRig(t, options{})
	u := r.signUp("ada@example.com")
	owner := r.as(r.ownerToken(u))

	r.expect(owner, http.MethodPost, "/v1/me/email-change", map[string]string{"email": "Grace@Example.com"}, http.StatusAccepted)
	code := r.mailedCode("grace@example.com")
	var got struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	r.expect(owner, http.MethodPost, "/v1/me/email-change/confirm", map[string]string{"code": code}, http.StatusOK).decode(t, &got)
	if got.ID != u.ID.String() || got.Email != "grace@example.com" {
		t.Errorf("confirmed %+v", got)
	}

	p := r.expect(owner, http.MethodPost, "/v1/me/email-change/confirm", map[string]string{"code": code}, http.StatusBadRequest).problem(t)
	if p.Errors["code"] == "" {
		t.Errorf("problem = %+v, want an error for code", p)
	}
	p = r.expect(owner, http.MethodPost, "/v1/me/email-change", map[string]string{"email": "grace@example.com"}, http.StatusBadRequest).problem(t)
	if p.Errors["email"] == "" {
		t.Errorf("problem = %+v, want an error for email", p)
	}
}

func TestConfirmEmailChangeToATakenAddress(t *testing.T) {
	r := newRig(t, options{})
	u := r.signUp("ada@example.com")
	owner := r.as(r.ownerToken(u))
	r.expect(owner, http.MethodPost, "/v1/me/email-change", map[string]string{"email": "grace@example.com"}, http.StatusAccepted)
	code := r.mailedCode("grace@example.com")
	r.signUp("grace@example.com")

	p := r.expect(owner, http.MethodPost, "/v1/me/email-change/confirm", map[string]string{"code": code}, http.StatusConflict).problem(t)
	if p.Detail != "This email address belongs to another account." {
		t.Errorf("detail = %q", p.Detail)
	}
}

func TestCrossOriginWritesAreRefused(t *testing.T) {
	r := newRig(t, options{})
	u := r.signUp("ada@example.com")
	cookie := header{"Cookie": "access_token=" + r.ownerToken(u), "Sec-Fetch-Site": "cross-site"}
	for _, path := range []string{"/v1/login-links", "/v1/me/email-change"} {
		res := r.do(r.browser(), http.MethodPost, path, map[string]string{"email": "eve@example.com"}, cookie)
		if res.status != http.StatusForbidden {
			t.Errorf("cross-site POST %s = %d, want 403", path, res.status)
		}
	}
	if links := r.mailed(); len(links) != 0 {
		t.Errorf("mailed %+v", links)
	}
}

func TestUnknownRoute(t *testing.T) {
	r := newRig(t, options{})
	res, err := r.browser().client.Get(r.srv.URL + "/v1/users")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound || res.Header.Get("Content-Type") != "application/problem+json" {
		t.Errorf("GET /v1/users = %d %s, want a 404 problem", res.StatusCode, res.Header.Get("Content-Type"))
	}
}

func TestSecurityHeaders(t *testing.T) {
	r := newRig(t, options{})
	res := r.expect(r.browser(), http.MethodGet, "/v1/config", nil, http.StatusOK)
	for k, want := range map[string]string{"X-Content-Type-Options": "nosniff", "Cache-Control": "no-store"} {
		if got := res.header.Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	if res.header.Get("X-Request-Id") == "" {
		t.Error("no X-Request-Id")
	}
}
