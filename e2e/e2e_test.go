//go:build e2e

// Package e2e drives the whole stack through the gateway, the way a browser
// and a third-party app would. Start it first (make up, with any STORE),
// then: go test -tags e2e ./e2e/...
package e2e

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

var (
	baseURL    = env("E2E_BASE_URL", "http://localhost:8080")
	mailpitURL = env("E2E_MAILPIT_URL", "http://localhost:8025")
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// browser is a client with its own cookie jar that doesn't follow
// redirects, so tests can assert where each one points.
type browser struct {
	t      *testing.T
	client *http.Client
}

func newBrowser(t *testing.T) *browser {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &browser{t: t, client: &http.Client{
		Jar:           jar,
		Timeout:       10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

type response struct {
	status int
	header http.Header
	body   []byte
}

func (r response) decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.body, v); err != nil {
		t.Fatalf("decode %s: %v", r.body, err)
	}
}

func (b *browser) do(method, path string, body any, header http.Header) response {
	b.t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			b.t.Fatal(err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(b.t.Context(), method, baseURL+path, reader)
	if err != nil {
		b.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range header {
		req.Header[k] = v
	}
	res, err := b.client.Do(req)
	if err != nil {
		b.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		b.t.Fatal(err)
	}
	return response{status: res.StatusCode, header: res.Header, body: data}
}

func (b *browser) expect(method, path string, body any, want int) response {
	b.t.Helper()
	res := b.do(method, path, body, nil)
	if res.status != want {
		b.t.Fatalf("%s %s = %d, want %d: %s", method, path, res.status, want, res.body)
	}
	return res
}

func newEmail(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("e2e-%s@example.com", strings.ToLower(rand.Text()[:12]))
}

var codeInLink = regexp.MustCompile(`[?&]code=([A-Za-z0-9_-]+)`)

// mailedCode waits for the newest email to address and returns the code in
// its link. Mail goes through the outbox, so it arrives shortly after the
// request that caused it.
func mailedCode(t *testing.T, address string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for {
		var found struct {
			Messages []struct{ ID string } `json:"messages"`
		}
		getJSON(t, ctx, mailpitURL+"/api/v1/search?query="+url.QueryEscape("to:"+address), &found)
		if len(found.Messages) > 0 {
			var msg struct{ Text string }
			getJSON(t, ctx, mailpitURL+"/api/v1/message/"+found.Messages[0].ID, &msg)
			m := codeInLink.FindStringSubmatch(msg.Text)
			if m == nil {
				t.Fatalf("the email to %s has no code link:\n%s", address, msg.Text)
			}
			deleteMail(t, found.Messages[0].ID)
			return m[1]
		}
		select {
		case <-ctx.Done():
			t.Fatalf("no email to %s arrived in Mailpit", address)
		case <-tick.C:
		}
	}
}

func getJSON(t *testing.T, ctx context.Context, rawURL string, v any) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", rawURL, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %d", rawURL, res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(v); err != nil {
		t.Fatalf("GET %s: %v", rawURL, err)
	}
}

func deleteMail(t *testing.T, id string) {
	t.Helper()
	data, _ := json.Marshal(map[string][]string{"IDs": {id}})
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, mailpitURL+"/api/v1/messages", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if res, err := http.DefaultClient.Do(req); err == nil {
		res.Body.Close()
	}
}

// signIn requests a login link for email, follows it from Mailpit and
// returns the signed-in browser.
func signIn(t *testing.T, email string) *browser {
	t.Helper()
	b := newBrowser(t)
	b.expect(http.MethodPost, "/api/user/v1/login-links", map[string]string{"email": email}, http.StatusAccepted)
	b.expect(http.MethodPost, "/api/auth/v1/sessions", map[string]string{"code": mailedCode(t, email)}, http.StatusCreated)
	return b
}

type me struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	GoogleLinked bool   `json:"google_linked"`
}

func TestSignInWithEmail(t *testing.T) {
	email := newEmail(t)
	b := newBrowser(t)
	b.expect(http.MethodGet, "/api/user/v1/me", nil, http.StatusUnauthorized)

	b.expect(http.MethodPost, "/api/user/v1/login-links", map[string]string{"email": email}, http.StatusAccepted)
	code := mailedCode(t, email)
	b.expect(http.MethodPost, "/api/auth/v1/sessions", map[string]string{"code": code}, http.StatusCreated)

	var got me
	b.expect(http.MethodGet, "/api/user/v1/me", nil, http.StatusOK).decode(t, &got)
	if got.Email != email || got.ID == "" {
		t.Fatalf("me = %+v, want %s", got, email)
	}

	t.Run("a code works once", func(t *testing.T) {
		newBrowser(t).expect(http.MethodPost, "/api/auth/v1/sessions", map[string]string{"code": code}, http.StatusUnauthorized)
	})

	t.Run("signing in again finds the same user", func(t *testing.T) {
		var again me
		signIn(t, strings.ToUpper(email)).expect(http.MethodGet, "/api/user/v1/me", nil, http.StatusOK).decode(t, &again)
		if again.ID != got.ID {
			t.Errorf("second sign-in is user %s, want %s", again.ID, got.ID)
		}
	})

	t.Run("refresh rotates the session", func(t *testing.T) {
		b.expect(http.MethodPost, "/api/auth/v1/sessions/refresh", nil, http.StatusOK)
		b.expect(http.MethodGet, "/api/user/v1/me", nil, http.StatusOK)
	})

	t.Run("sign out ends the session", func(t *testing.T) {
		b.expect(http.MethodDelete, "/api/auth/v1/sessions/current", nil, http.StatusNoContent)
		b.expect(http.MethodGet, "/api/user/v1/me", nil, http.StatusUnauthorized)
		b.expect(http.MethodPost, "/api/auth/v1/sessions/refresh", nil, http.StatusUnauthorized)
	})
}

func TestRevokeAnotherSession(t *testing.T) {
	email := newEmail(t)
	laptop := signIn(t, email)
	phone := signIn(t, email)

	var list struct {
		Sessions []struct {
			ID      string `json:"id"`
			Current bool   `json:"current"`
		} `json:"sessions"`
	}
	laptop.expect(http.MethodGet, "/api/auth/v1/sessions", nil, http.StatusOK).decode(t, &list)
	if len(list.Sessions) != 2 {
		t.Fatalf("listed %d sessions, want 2", len(list.Sessions))
	}
	var other string
	for _, s := range list.Sessions {
		if !s.Current {
			other = s.ID
		}
	}
	if other == "" {
		t.Fatalf("no session other than the current one: %+v", list.Sessions)
	}

	laptop.expect(http.MethodDelete, "/api/auth/v1/sessions/"+other, nil, http.StatusNoContent)
	phone.expect(http.MethodPost, "/api/auth/v1/sessions/refresh", nil, http.StatusUnauthorized)
	laptop.expect(http.MethodGet, "/api/user/v1/me", nil, http.StatusOK)

	t.Run("nobody else can revoke it", func(t *testing.T) {
		var mine struct {
			Sessions []struct{ ID string } `json:"sessions"`
		}
		laptop.expect(http.MethodGet, "/api/auth/v1/sessions", nil, http.StatusOK).decode(t, &mine)
		stranger := signIn(t, newEmail(t))
		stranger.expect(http.MethodDelete, "/api/auth/v1/sessions/"+mine.Sessions[0].ID, nil, http.StatusNotFound)
	})
}

func TestChangeEmail(t *testing.T) {
	email, next := newEmail(t), newEmail(t)
	b := signIn(t, email)

	b.expect(http.MethodPost, "/api/user/v1/me/email-change", map[string]string{"email": next}, http.StatusAccepted)
	b.expect(http.MethodPost, "/api/user/v1/me/email-change/confirm", map[string]string{"code": mailedCode(t, next)}, http.StatusOK)

	var got me
	b.expect(http.MethodGet, "/api/user/v1/me", nil, http.StatusOK).decode(t, &got)
	if got.Email != next {
		t.Fatalf("email = %s, want %s", got.Email, next)
	}
}

type client struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
}

func createClient(t *testing.T, b *browser, scopes ...string) (client, string) {
	t.Helper()
	var created struct {
		Client       client `json:"client"`
		ClientSecret string `json:"client_secret"`
	}
	b.expect(http.MethodPost, "/api/auth/v1/clients", map[string]any{
		"name":          "e2e app",
		"redirect_uris": []string{"http://localhost:9999/callback"},
		"scopes":        scopes,
	}, http.StatusCreated).decode(t, &created)
	if created.Client.ID == "" || created.ClientSecret == "" {
		t.Fatalf("created %+v", created)
	}
	return created.Client, created.ClientSecret
}

func TestOAuthClientCredentials(t *testing.T) {
	owner := signIn(t, newEmail(t))
	c, secret := createClient(t, owner, "user:read")

	var list struct {
		Clients []json.RawMessage `json:"clients"`
	}
	res := owner.expect(http.MethodGet, "/api/auth/v1/clients", nil, http.StatusOK)
	res.decode(t, &list)
	if len(list.Clients) != 1 || bytes.Contains(res.body, []byte(secret)) {
		t.Fatalf("client list = %s: want one client and no secret", res.body)
	}

	cfg := clientcredentials.Config{
		ClientID:     c.ID,
		ClientSecret: secret,
		TokenURL:     baseURL + "/api/auth/oauth/token",
		Scopes:       []string{"user:read"},
	}
	token, err := cfg.Token(t.Context())
	if err != nil {
		t.Fatalf("client credentials grant: %v", err)
	}

	t.Run("a client token is not a user", func(t *testing.T) {
		b := newBrowser(t)
		res := b.do(http.MethodGet, "/api/user/v1/me", nil, http.Header{"Authorization": {"Bearer " + token.AccessToken}})
		if res.status != http.StatusForbidden {
			t.Fatalf("GET /me with a client token = %d, want 403: %s", res.status, res.body)
		}
	})

	t.Run("a wrong secret is refused", func(t *testing.T) {
		wrong := cfg
		wrong.ClientSecret = "not-the-secret"
		if _, err := wrong.Token(t.Context()); err == nil {
			t.Fatal("got a token with the wrong secret")
		}
	})

	t.Run("a scope the client doesn't have is refused", func(t *testing.T) {
		wider := cfg
		wider.Scopes = []string{"clients:write"}
		if _, err := wider.Token(t.Context()); err == nil {
			t.Fatal("got a token for a scope the client wasn't registered with")
		}
	})

	t.Run("deleting the client stops its credentials", func(t *testing.T) {
		owner.expect(http.MethodDelete, "/api/auth/v1/clients/"+c.ID, nil, http.StatusNoContent)
		if _, err := cfg.Token(t.Context()); err == nil {
			t.Fatal("a deleted client still gets tokens")
		}
	})
}

func TestOAuthAuthorizationCodeWithPKCE(t *testing.T) {
	email := newEmail(t)
	user := signIn(t, email)
	c, secret := createClient(t, user, "user:read")

	cfg := oauth2.Config{
		ClientID:     c.ID,
		ClientSecret: secret,
		RedirectURL:  c.RedirectURIs[0],
		Scopes:       []string{"user:read"},
		Endpoint: oauth2.Endpoint{
			AuthURL:   baseURL + "/api/auth/oauth/authorize",
			TokenURL:  baseURL + "/api/auth/oauth/token",
			AuthStyle: oauth2.AuthStyleInHeader,
		},
	}
	verifier := oauth2.GenerateVerifier()
	authorize := strings.TrimPrefix(cfg.AuthCodeURL("e2e-state", oauth2.S256ChallengeOption(verifier)), baseURL)

	t.Run("without a session the user is sent to sign in", func(t *testing.T) {
		res := newBrowser(t).expect(http.MethodGet, authorize, nil, http.StatusFound)
		if loc := res.header.Get("Location"); !strings.Contains(loc, "/sign-in?") {
			t.Fatalf("redirected to %s, want the sign-in page", loc)
		}
	})

	res := user.expect(http.MethodGet, authorize, nil, http.StatusFound)
	consent, err := url.Parse(res.header.Get("Location"))
	if err != nil || consent.Path != "/consent" || consent.Query().Get("request") == "" {
		t.Fatalf("redirected to %s, want the consent page", res.header.Get("Location"))
	}
	request := consent.Query().Get("request")

	var details struct {
		Client struct{ Name string } `json:"client"`
		Scopes []struct{ Name string } `json:"scopes"`
	}
	user.expect(http.MethodGet, "/api/auth/v1/authorization-requests/"+request, nil, http.StatusOK).decode(t, &details)
	if details.Client.Name != "e2e app" || len(details.Scopes) != 1 || details.Scopes[0].Name != "user:read" {
		t.Fatalf("consent details = %+v", details)
	}

	form := url.Values{"request": {request}, "decision": {"allow"}}
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, baseURL+"/api/auth/oauth/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	allowed, err := user.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	allowed.Body.Close()
	if allowed.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST /oauth/authorize = %d, want 303", allowed.StatusCode)
	}
	back, _ := url.Parse(allowed.Header.Get("Location"))
	if back.Query().Get("state") != "e2e-state" || back.Query().Get("code") == "" {
		t.Fatalf("redirected back to %s, want a code and the state", back)
	}

	t.Run("the code needs the verifier", func(t *testing.T) {
		if _, err := cfg.Exchange(t.Context(), back.Query().Get("code"), oauth2.VerifierOption("wrong-verifier-wrong-verifier-wrong-verifier")); err == nil {
			t.Fatal("exchanged a code with the wrong verifier")
		}
	})

	// A failed exchange spends the code, so ask for a fresh one.
	res = user.expect(http.MethodGet, authorize, nil, http.StatusFound)
	consent, _ = url.Parse(res.header.Get("Location"))
	form.Set("request", consent.Query().Get("request"))
	req, _ = http.NewRequestWithContext(t.Context(), http.MethodPost, baseURL+"/api/auth/oauth/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	allowed, err = user.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	allowed.Body.Close()
	back, _ = url.Parse(allowed.Header.Get("Location"))

	token, err := cfg.Exchange(t.Context(), back.Query().Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if token.RefreshToken == "" {
		t.Fatal("no refresh token")
	}

	var got me
	app := newBrowser(t)
	r := app.do(http.MethodGet, "/api/user/v1/me", nil, http.Header{"Authorization": {"Bearer " + token.AccessToken}})
	if r.status != http.StatusOK {
		t.Fatalf("GET /me with the app's token = %d: %s", r.status, r.body)
	}
	r.decode(t, &got)
	if got.Email != email {
		t.Fatalf("the app sees %s, want %s", got.Email, email)
	}

	t.Run("refresh tokens rotate and a reused one revokes the grant", func(t *testing.T) {
		expired := *token
		expired.Expiry = time.Now().Add(-time.Minute)
		fresh, err := cfg.TokenSource(t.Context(), &expired).Token()
		if err != nil {
			t.Fatalf("refresh: %v", err)
		}
		if fresh.RefreshToken == token.RefreshToken {
			t.Fatal("the refresh token didn't rotate")
		}
		if _, err := cfg.TokenSource(t.Context(), &expired).Token(); err == nil {
			t.Fatal("a rotated refresh token worked again")
		}
		stale := *fresh
		stale.Expiry = time.Now().Add(-time.Minute)
		if _, err := cfg.TokenSource(t.Context(), &stale).Token(); err == nil {
			t.Fatal("the grant survived refresh token reuse")
		}
	})
}
