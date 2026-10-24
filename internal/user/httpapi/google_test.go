package httpapi_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/user/httpapi"
	"github.com/vardius/go-api-boilerplate/internal/user/store"
)

var ada = account{sub: "1084", email: "Ada@Gmail.com", verified: true}

// start begins a Google sign-in in b and returns Google's consent page.
func (r *rig) start(b *browser, query string) *url.URL {
	r.t.Helper()
	return r.expect(b, http.MethodGet, "/v1/google/start"+query, nil, http.StatusFound).location(r.t)
}

// callback brings b back from Google with q and returns where it is sent.
func (r *rig) callback(b *browser, q url.Values) *url.URL {
	r.t.Helper()
	return r.expect(b, http.MethodGet, "/v1/google/callback?"+q.Encode(), nil, http.StatusFound).location(r.t)
}

// googleSignIn signs a in through Google in a new browser and returns where
// the browser ends up.
func (r *rig) googleSignIn(a account, query string) *url.URL {
	r.t.Helper()
	b := r.browser()
	return r.callback(b, r.google.consent(r.start(b, query), a))
}

func (r *rig) redeem(signedIn *url.URL) store.User {
	r.t.Helper()
	if got := signedIn.Scheme + "://" + signedIn.Host + signedIn.Path; got != webURL+"/sign-in/complete" {
		r.t.Fatalf("sent to %s, want the sign-in completion page", signedIn)
	}
	u, err := r.app.RedeemLoginCode(authService, signedIn.Query().Get("code"))
	if err != nil {
		r.t.Fatal(err)
	}
	return u
}

func assertSentTo(t *testing.T, got *url.URL, want string) {
	t.Helper()
	if got.String() != want {
		t.Errorf("sent to %s, want %s", got, want)
	}
}

func TestGoogleIsOffWithoutAClient(t *testing.T) {
	r := newRig(t, options{})
	for _, path := range []string{"/v1/google/start", "/v1/google/callback?code=x&state=y"} {
		p := r.expect(r.browser(), http.MethodGet, path, nil, http.StatusNotFound).problem(t)
		if p.Detail != "Google sign-in is not set up on this server." {
			t.Errorf("GET %s detail = %q", path, p.Detail)
		}
	}
}

func TestGoogleSignInRegisters(t *testing.T) {
	r := newRig(t, options{google: true})
	b := r.browser()
	res := r.expect(b, http.MethodGet, "/v1/google/start?next=%2Fconsent%3Frequest%3D42&state=browser-state_1", nil, http.StatusFound)

	cookie := res.header.Get("Set-Cookie")
	for _, attr := range []string{"google_flow=", "Path=/api/user/v1/google", "Max-Age=600", "HttpOnly", "SameSite=Lax"} {
		if !strings.Contains(cookie, attr) {
			t.Errorf("Set-Cookie = %q, want %s", cookie, attr)
		}
	}
	if strings.Contains(cookie, "Secure") {
		t.Errorf("Set-Cookie = %q: Secure over a plain-http callback", cookie)
	}
	consent := res.location(t)
	if got := consent.Query().Get("redirect_uri"); got != "http://localhost:8080/api/user/v1/google/callback" {
		t.Errorf("redirect_uri = %s", got)
	}

	// The jar keeps the cookie for the gateway's path; this server has no
	// prefix, so send it by hand.
	q := r.google.consent(consent, ada)
	cb := r.do(b, http.MethodGet, "/v1/google/callback?"+q.Encode(), nil, header{"Cookie": strings.Split(cookie, ";")[0]})
	if cb.status != http.StatusFound {
		t.Fatalf("callback = %d: %s", cb.status, cb.body)
	}
	if cleared := cb.header.Get("Set-Cookie"); !strings.Contains(cleared, "google_flow=;") || !strings.Contains(cleared, "Max-Age=0") {
		t.Errorf("the callback left the flow cookie: Set-Cookie = %q", cleared)
	}
	to := cb.location(t)
	if next := to.Query().Get("next"); next != "/consent?request=42" {
		t.Errorf("next = %q", next)
	}
	// The web app's state comes back, so the browser that started knows the link is its own.
	if state := to.Query().Get("state"); state != "browser-state_1" {
		t.Errorf("state = %q, want the one the browser started with", state)
	}
	u := r.redeem(to)
	if u.Email != "ada@gmail.com" || u.GoogleSubject != "1084" {
		t.Errorf("signed in %+v", u)
	}

	// The same flow can't be finished twice.
	assertSentTo(t, r.callback(b, q), webURL+"/sign-in?google=expired")
}

func TestGoogleSignInSecureCookie(t *testing.T) {
	r := newRig(t, options{google: true, redirectURL: "https://example.com/api/user/v1/google/callback"})
	res := r.expect(r.browser(), http.MethodGet, "/v1/google/start", nil, http.StatusFound)
	if cookie := res.header.Get("Set-Cookie"); !strings.Contains(cookie, "Secure") {
		t.Errorf("Set-Cookie = %q, want Secure", cookie)
	}
}

func TestGoogleSignInFindsTheAccount(t *testing.T) {
	r := newRig(t, options{google: true, redirectURL: "http://localhost/v1/google/callback"})
	existing := r.signUp("ada@gmail.com")

	first := r.redeem(r.googleSignIn(ada, ""))
	again := r.redeem(r.googleSignIn(account{sub: "1084", email: "renamed@gmail.com", verified: true}, ""))
	if first.ID != existing.ID || again.ID != existing.ID || first.GoogleSubject != "1084" {
		t.Errorf("signed in %+v then %+v, want %s", first, again, existing.ID)
	}
}

func TestGoogleSignInFails(t *testing.T) {
	r := newRig(t, options{google: true, redirectURL: "http://localhost/v1/google/callback"})
	r.redeem(r.googleSignIn(ada, ""))
	before := r.google.exchanges()

	tests := []struct {
		name string
		// finish takes the browser that started and Google's consent page,
		// and returns where the callback sends it.
		finish func(b *browser, consent *url.URL) *url.URL
		want   string
	}{
		{"the user says no", func(b *browser, consent *url.URL) *url.URL {
			return r.callback(b, url.Values{"error": {"access_denied"}, "state": {consent.Query().Get("state")}})
		}, "denied"},
		{"Google reports an error", func(b *browser, consent *url.URL) *url.URL {
			return r.callback(b, url.Values{"error": {"server_error"}, "state": {consent.Query().Get("state")}})
		}, "failed"},
		{"another flow's state", func(b *browser, consent *url.URL) *url.URL {
			q := r.google.consent(consent, ada)
			q.Set("state", "forged")
			return r.callback(b, q)
		}, "expired"},
		{"no flow cookie", func(_ *browser, consent *url.URL) *url.URL {
			return r.callback(r.browser(), r.google.consent(consent, ada))
		}, "expired"},
		{"too slow", func(b *browser, consent *url.URL) *url.URL {
			q := r.google.consent(consent, ada)
			r.advance(10 * time.Minute)
			return r.callback(b, q)
		}, "expired"},
		{"an unverified address", func(b *browser, consent *url.URL) *url.URL {
			return r.callback(b, r.google.consent(consent, account{sub: "2048", email: "eve@gmail.com"}))
		}, "unverified"},
		{"a token for another sign-in", func(b *browser, consent *url.URL) *url.URL {
			return r.callback(b, r.google.consent(consent, account{sub: "2048", email: "eve@gmail.com", verified: true, nonce: "replayed"}))
		}, "failed"},
		{"a made-up code", func(b *browser, consent *url.URL) *url.URL {
			return r.callback(b, url.Values{"code": {"made-up"}, "state": {consent.Query().Get("state")}})
		}, "failed"},
		{"a second Google account for a linked address", func(b *browser, consent *url.URL) *url.URL {
			return r.callback(b, r.google.consent(consent, account{sub: "2048", email: "ada@gmail.com", verified: true}))
		}, "taken"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := r.browser()
			assertSentTo(t, tt.finish(b, r.start(b, "")), webURL+"/sign-in?google="+tt.want)
		})
	}
	if got := r.google.exchanges() - before; got != 4 {
		t.Errorf("exchanged %d codes, want 4: no exchange without a valid flow", got)
	}
}

// The cookie jar sends the flow cookie only under the callback's directory.
func TestGoogleFlowCookieStaysWithTheCallback(t *testing.T) {
	r := newRig(t, options{google: true, redirectURL: "http://localhost/v1/google/callback"})
	b := r.browser()
	r.start(b, "")
	srv, _ := url.Parse(r.srv.URL)
	for path, want := range map[string]bool{"/v1/google/callback": true, "/v1/me": false} {
		u := *srv
		u.Path = path
		got := len(b.client.Jar.Cookies(&u)) == 1
		if got != want {
			t.Errorf("cookie sent to %s: %v, want %v", path, got, want)
		}
	}
}

func TestGoogleStartRefuses(t *testing.T) {
	r := newRig(t, options{google: true})
	u := r.signUp("ada@example.com")
	tests := []struct {
		name   string
		b      *browser
		query  string
		status int
	}{
		{"linking anonymously", r.browser(), "?link=1", http.StatusUnauthorized},
		{"linking with a client token", r.as(r.clientToken()), "?link=1", http.StatusForbidden},
		{"linking without user:write", r.as(r.userToken(u.ID, "user:read")), "?link=1", http.StatusForbidden},
		{"a bad link flag", r.as(r.ownerToken(u)), "?link=true", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := r.expect(tt.b, http.MethodGet, "/v1/google/start"+tt.query, nil, tt.status)
			if res.header.Get("Set-Cookie") != "" {
				t.Error("set a flow cookie on a refused start")
			}
		})
	}
}

func TestGoogleStartWhileGoogleIsDown(t *testing.T) {
	r := newRig(t, options{google: true})
	r.google.setDown(true)
	r.expect(r.browser(), http.MethodGet, "/v1/google/start", nil, http.StatusInternalServerError)

	// Discovery is retried once Google is back.
	r.google.setDown(false)
	r.expect(r.browser(), http.MethodGet, "/v1/google/start", nil, http.StatusFound)
}

func TestLinkGoogle(t *testing.T) {
	r := newRig(t, options{google: true, redirectURL: "http://localhost/v1/google/callback"})
	u := r.signUp("ada@example.com")
	owner := r.as(r.ownerToken(u))

	q := r.google.consent(r.start(owner, "?link=1"), ada)
	assertSentTo(t, r.callback(owner, q), webURL+"/profile?google=linked")

	var me struct {
		GoogleLinked bool `json:"google_linked"`
	}
	r.expect(owner, http.MethodGet, "/v1/me", nil, http.StatusOK).decode(t, &me)
	if !me.GoogleLinked {
		t.Error("google_linked = false after linking")
	}
	// Signing in with Google now finds this user, whatever its address.
	if got := r.redeem(r.googleSignIn(ada, "")); got.ID != u.ID {
		t.Errorf("Google sign-in is user %s, want %s", got.ID, u.ID)
	}
}

func TestLinkGoogleFails(t *testing.T) {
	r := newRig(t, options{google: true, redirectURL: "http://localhost/v1/google/callback"})
	r.redeem(r.googleSignIn(ada, ""))
	eve := r.signUp("eve@example.com")
	grace := r.signUp("grace@example.com")

	tests := []struct {
		name string
		// at is who comes back from Google, having started as eve.
		at   *browser
		a    account
		want string
	}{
		{"a Google account linked to someone else", nil, ada, "taken"},
		{"another user comes back", r.as(r.ownerToken(grace)), account{sub: "4096", email: "x@gmail.com", verified: true}, "expired"},
		{"signed out on the way", r.browser(), account{sub: "4096", email: "x@gmail.com", verified: true}, "expired"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := r.as(r.ownerToken(eve))
			q := r.google.consent(r.start(b, "?link=1"), tt.a)
			back := b
			if tt.at != nil {
				back = tt.at
				// The flow cookie lives in eve's browser, whoever's token
				// it now carries.
				back.client.Jar = b.client.Jar
			}
			assertSentTo(t, r.callback(back, q), webURL+"/profile?google="+tt.want)
		})
	}
}

func TestNewGoogleRejectsBadConfig(t *testing.T) {
	good := httpapi.GoogleConfig{ClientID: "id", ClientSecret: "secret", RedirectURL: "https://example.com/api/user/v1/google/callback"}
	tests := []struct {
		name string
		edit func(c *httpapi.GoogleConfig)
	}{
		{"no secret", func(c *httpapi.GoogleConfig) { c.ClientSecret = "" }},
		{"no redirect", func(c *httpapi.GoogleConfig) { c.RedirectURL = "" }},
		{"a relative redirect", func(c *httpapi.GoogleConfig) { c.RedirectURL = "/api/user/v1/google/callback" }},
		{"a redirect elsewhere", func(c *httpapi.GoogleConfig) { c.RedirectURL = "https://example.com/callback" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := good
			tt.edit(&cfg)
			if _, err := httpapi.NewGoogle(cfg, httpapi.GoogleIssuer, http.DefaultClient); err == nil {
				t.Error("NewGoogle accepted it")
			}
		})
	}
	if _, err := httpapi.NewGoogle(good, httpapi.GoogleIssuer, http.DefaultClient); err != nil {
		t.Errorf("NewGoogle(good) = %v", err)
	}
}
