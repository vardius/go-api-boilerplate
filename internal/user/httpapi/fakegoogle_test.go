package httpapi_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/coreos/go-oidc/v3/oidc/oidctest"
)

const (
	googleClientID     = "web-client.apps.googleusercontent.com"
	googleClientSecret = "google-secret"
)

// fakeGoogle is an OpenID provider: discovery and keys from oidctest, and a
// token endpoint that hands out the ID token a test approved, once, to the
// right client with the right PKCE verifier.
type fakeGoogle struct {
	t   *testing.T
	srv *httptest.Server
	key *ecdsa.PrivateKey
	now func() time.Time

	mu       sync.Mutex
	down     bool
	approved map[string]grant // by code
	exchange int              // how many codes were exchanged
}

// grant is what the user consented to at the fake.
type grant struct {
	challenge   string
	redirectURI string
	claims      map[string]any
}

func newFakeGoogle(t *testing.T, now func() time.Time) *fakeGoogle {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	g := &fakeGoogle{t: t, key: key, now: now, approved: map[string]grant{}}
	discovery := &oidctest.Server{
		PublicKeys: []oidctest.PublicKey{{PublicKey: key.Public(), KeyID: "k1", Algorithm: oidc.ES256}},
		Algorithms: []string{oidc.ES256},
	}
	mux := http.NewServeMux()
	mux.Handle("/", discovery)
	mux.HandleFunc("POST /token", g.token)
	g.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.mu.Lock()
		down := g.down
		g.mu.Unlock()
		if down {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(g.srv.Close)
	discovery.SetIssuer(g.srv.URL)
	return g
}

func (g *fakeGoogle) issuer() string       { return g.srv.URL }
func (g *fakeGoogle) client() *http.Client { return g.srv.Client() }
func (g *fakeGoogle) authURL() string      { return g.srv.URL + "/auth" }

func (g *fakeGoogle) setDown(down bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.down = down
}

func (g *fakeGoogle) exchanges() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.exchange
}

// account is who signs in at the fake.
type account struct {
	sub      string
	email    string
	verified bool
	// nonce, when set, replaces the nonce the sign-in asked for.
	nonce string
}

// consent plays the user at Google's consent page, which consent's URL is:
// it approves account and returns the callback query Google redirects
// with.
func (g *fakeGoogle) consent(consentURL *url.URL, a account) url.Values {
	g.t.Helper()
	q := consentURL.Query()
	if got := consentURL.Scheme + "://" + consentURL.Host + consentURL.Path; got != g.authURL() {
		g.t.Fatalf("sent to %s, want %s", got, g.authURL())
	}
	want := map[string]string{
		"client_id":             googleClientID,
		"response_type":         "code",
		"scope":                 "openid email",
		"code_challenge_method": "S256",
	}
	for k, v := range want {
		if q.Get(k) != v {
			g.t.Errorf("consent URL %s = %q, want %q", k, q.Get(k), v)
		}
	}
	for _, k := range []string{"state", "nonce", "code_challenge", "redirect_uri"} {
		if q.Get(k) == "" {
			g.t.Fatalf("consent URL has no %s: %s", k, consentURL)
		}
	}
	nonce := q.Get("nonce")
	if a.nonce != "" {
		nonce = a.nonce
	}
	now := g.now()
	code := rand.Text()
	g.mu.Lock()
	g.approved[code] = grant{
		challenge:   q.Get("code_challenge"),
		redirectURI: q.Get("redirect_uri"),
		claims: map[string]any{
			"iss":            g.srv.URL,
			"aud":            googleClientID,
			"sub":            a.sub,
			"email":          a.email,
			"email_verified": a.verified,
			"nonce":          nonce,
			"iat":            now.Unix(),
			"exp":            now.Add(time.Hour).Unix(),
		},
	}
	g.mu.Unlock()
	return url.Values{"code": {code}, "state": {q.Get("state")}}
}

func (g *fakeGoogle) token(w http.ResponseWriter, r *http.Request) {
	fail := func(code string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.MarshalWrite(w, map[string]string{"error": code})
	}
	if err := r.ParseForm(); err != nil {
		fail("invalid_request")
		return
	}
	id, secret, ok := r.BasicAuth()
	if !ok {
		id, secret = r.PostForm.Get("client_id"), r.PostForm.Get("client_secret")
	}
	if id != googleClientID || secret != googleClientSecret {
		fail("invalid_client")
		return
	}
	code := r.PostForm.Get("code")
	g.mu.Lock()
	gr, ok := g.approved[code]
	delete(g.approved, code)
	g.exchange++
	g.mu.Unlock()
	sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
	if !ok || r.PostForm.Get("grant_type") != "authorization_code" || r.PostForm.Get("redirect_uri") != gr.redirectURI ||
		base64.RawURLEncoding.EncodeToString(sum[:]) != gr.challenge {
		fail("invalid_grant")
		return
	}
	claims, err := json.Marshal(gr.claims)
	if err != nil {
		g.t.Error(err)
		fail("server_error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.MarshalWrite(w, map[string]any{
		"access_token": "google-access-token",
		"token_type":   "Bearer",
		"expires_in":   3600,
		"id_token":     oidctest.SignIDToken(g.key, "k1", oidc.ES256, string(claims)),
	})
}
