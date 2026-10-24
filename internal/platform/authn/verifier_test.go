package authn_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn/authntest"
)

// clock is a settable time for the Verifier, safe to read from its
// background fetches.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func newClock(v *authn.Verifier) *clock {
	c := &clock{now: time.Now()}
	v.Now = c.Now
	return c
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestVerifyUserToken(t *testing.T) {
	iss := authntest.NewIssuer(t)
	user := uuid.NewV7()
	token := iss.Token(t, authn.Identity{UserID: user, Scopes: []string{"user:read", "user:write"}})

	id, err := iss.Verifier().Verify(t.Context(), token)
	if err != nil {
		t.Fatal(err)
	}
	if id.UserID != user || id.Subject != user.String() || id.ClientID != "web" || !slices.Equal(id.Scopes, []string{"user:read", "user:write"}) {
		t.Errorf("identity = %+v", id)
	}
	if !id.IsUser() || !id.IsFirstParty() || !id.Has("user:write") || id.Has("clients:read") {
		t.Errorf("predicates wrong for %+v", id)
	}
}

func TestVerifyClientToken(t *testing.T) {
	iss := authntest.NewIssuer(t)
	token := iss.Token(t, authn.Identity{ClientID: "auth-service", Scopes: []string{"login-codes:redeem"}})

	id, err := iss.Verifier().Verify(t.Context(), token)
	if err != nil {
		t.Fatal(err)
	}
	if id.UserID != uuid.Nil() || id.Subject != "auth-service" || id.IsUser() || id.IsFirstParty() || !id.Has("login-codes:redeem") {
		t.Errorf("identity = %+v", id)
	}
}

func TestVerifyAllowsClockSkew(t *testing.T) {
	iss := authntest.NewIssuer(t)
	token := iss.Token(t, authn.Identity{UserID: uuid.NewV7()}, authntest.WithExpiry(-20*time.Second))
	if _, err := iss.Verifier().Verify(t.Context(), token); err != nil {
		t.Errorf("a token 20s past expiry, inside the 30s leeway, failed: %v", err)
	}
}

// forged signs claims that would otherwise pass, the way an attacker who
// only has the public JWKS could.
func forged(t *testing.T, iss *authntest.Issuer, method jwt.SigningMethod, key any) string {
	t.Helper()
	cfg := iss.Config()
	valid, _, err := jwt.NewParser().ParseUnverified(iss.Token(t, authn.Identity{UserID: uuid.NewV7()}), jwt.MapClaims{})
	if err != nil {
		t.Fatal(err)
	}
	token := jwt.NewWithClaims(method, jwt.MapClaims{
		"iss":       cfg.Issuer,
		"aud":       cfg.Audience,
		"sub":       uuid.NewV7().String(),
		"client_id": "web",
		"scope":     "user:read user:write clients:write",
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	})
	token.Header["typ"] = "at+jwt"
	token.Header["kid"] = valid.Header["kid"]
	s, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func jwksBytes(t *testing.T, iss *authntest.Issuer) []byte {
	t.Helper()
	b, err := fetchJWKS(iss)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func fetchJWKS(iss *authntest.Issuer) ([]byte, error) {
	res, err := http.Get(iss.Config().JWKSURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

func TestVerifyRejects(t *testing.T) {
	iss := authntest.NewIssuer(t)
	other, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	user := authn.Identity{UserID: uuid.NewV7(), Scopes: []string{"user:read"}}

	tests := []struct {
		name  string
		token string
	}{
		{"expired", iss.Token(t, user, authntest.WithExpiry(-time.Minute))},
		{"wrong audience", iss.Token(t, user, authntest.WithAudience("someone-else"))},
		{"wrong issuer", iss.Token(t, user, authntest.WithIssuer("https://evil.example"))},
		{"no expiry", iss.Token(t, user, authntest.WithClaim("exp", nil))},
		{"no issuer", iss.Token(t, user, authntest.WithClaim("iss", nil))},
		{"no audience", iss.Token(t, user, authntest.WithClaim("aud", nil))},
		{"issued in the future", iss.Token(t, user, authntest.WithClaim("iat", time.Now().Add(time.Hour).Unix()))},
		{"alg none", forged(t, iss, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType)},
		{"HS256 keyed with the public JWKS", forged(t, iss, jwt.SigningMethodHS256, jwksBytes(t, iss))},
		{"ID token type", iss.Token(t, user, authntest.WithType("JWT"))},
		{"no type", iss.Token(t, user, authntest.WithType(""))},
		{"no kid", iss.Token(t, user, authntest.WithKID(""))},
		{"signed by another key", iss.Token(t, user, authntest.WithKey(other))},
		{"unknown kid", iss.Token(t, user, authntest.WithKey(other), authntest.WithKID("not-published"))},
		{"no client_id", iss.Token(t, user, authntest.WithClaim("client_id", nil))},
		{"no subject", iss.Token(t, user, authntest.WithClaim("sub", nil))},
		{"subject is neither user nor client", iss.Token(t, user, authntest.WithClaim("sub", "ada@example.com"))},
		{"nil user ID", iss.Token(t, user, authntest.WithClaim("sub", uuid.Nil().String()))},
		{"not a JWT", "not-a-token"},
		{"empty", ""},
	}
	v := iss.Verifier()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := v.Verify(t.Context(), tt.token)
			if !errors.Is(err, apperr.ErrUnauthorized) {
				t.Errorf("err = %v, want ErrUnauthorized", err)
			}
			if id.Subject != "" || id.Scopes != nil {
				t.Errorf("returned an identity with the error: %+v", id)
			}
			if apperr.Message(err) != "" {
				t.Errorf("the reason would reach the client: %q", apperr.Message(err))
			}
		})
	}
}

func TestVerifyPicksUpRotatedKeysAtMostOncePerMinute(t *testing.T) {
	iss := authntest.NewIssuer(t)
	v := iss.Verifier()
	clk := newClock(v)
	user := authn.Identity{UserID: uuid.NewV7()}
	verify := func(token string) error {
		_, err := v.Verify(t.Context(), token)
		return err
	}

	if err := verify(iss.Token(t, user)); err != nil {
		t.Fatal(err)
	}
	iss.Rotate(t)
	rotated := iss.Token(t, user)
	if err := verify(rotated); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Errorf("a new kid seconds after a fetch = %v, want a refusal", err)
	}
	for range 5 {
		_ = verify(iss.Token(t, user, authntest.WithKID(uuid.NewV7().String())))
	}
	if got := iss.JWKSRequests(); got != 1 {
		t.Errorf("JWKS fetched %d times within a minute, want 1", got)
	}

	clk.advance(time.Minute)
	if err := verify(rotated); err != nil {
		t.Errorf("the rotated key wasn't picked up a minute later: %v", err)
	}
	if err := verify(iss.Token(t, user, authntest.WithKID("made-up"))); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Errorf("made-up kid = %v", err)
	}
	if got := iss.JWKSRequests(); got != 2 {
		t.Errorf("JWKS fetched %d times, want 2", got)
	}
}

func TestVerifyRefreshesAnHourOldKeySet(t *testing.T) {
	iss := authntest.NewIssuer(t)
	v := iss.Verifier()
	clk := newClock(v)
	user := authn.Identity{UserID: uuid.NewV7()}
	// The tokens outlive the hour the clock moves on.
	old := iss.Token(t, user, authntest.WithExpiry(2*time.Hour))
	if _, err := v.Verify(t.Context(), old); err != nil {
		t.Fatal(err)
	}

	iss.Rotate(t)
	clk.advance(time.Hour)
	// The cached key still answers at once; the refetch runs behind it.
	if _, err := v.Verify(t.Context(), old); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Verify(t.Context(), iss.Token(t, user, authntest.WithExpiry(2*time.Hour))); err != nil {
		t.Errorf("the refreshed set lacks the new key: %v", err)
	}
	if got := iss.JWKSRequests(); got != 2 {
		t.Errorf("JWKS fetched %d times, want 2", got)
	}
}

func TestVerifyFetchesOnceForConcurrentCallers(t *testing.T) {
	iss := authntest.NewIssuer(t)
	v := iss.Verifier()
	token := iss.Token(t, authn.Identity{UserID: uuid.NewV7()})

	var wg sync.WaitGroup
	var failures atomic.Int32
	for range 20 {
		wg.Go(func() {
			if _, err := v.Verify(t.Context(), token); err != nil {
				failures.Add(1)
			}
		})
	}
	wg.Wait()
	if failures.Load() != 0 || iss.JWKSRequests() != 1 {
		t.Errorf("%d failures, %d fetches; want 0 and 1", failures.Load(), iss.JWKSRequests())
	}
}

// jwksProxy serves iss's JWKS through edit, or fails while failing is set.
type jwksProxy struct {
	iss     *authntest.Issuer
	edit    func(keys []any) []any
	failing atomic.Bool
}

func (p *jwksProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p.failing.Load() {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	var set struct {
		Keys []any `json:"keys"`
	}
	b, err := fetchJWKS(p.iss)
	if err == nil {
		err = json.Unmarshal(b, &set)
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if p.edit != nil {
		set.Keys = p.edit(set.Keys)
	}
	_ = json.NewEncoder(w).Encode(set)
}

func verifierFor(t *testing.T, iss *authntest.Issuer, h http.Handler) *authn.Verifier {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	cfg := iss.Config()
	cfg.JWKSURL = srv.URL
	return authn.NewVerifier(cfg, nil)
}

func TestVerifySkipsKeysItCantUse(t *testing.T) {
	iss := authntest.NewIssuer(t)
	v := verifierFor(t, iss, &jwksProxy{iss: iss, edit: func(keys []any) []any {
		good := keys[0].(map[string]any)
		with := func(k, val string) map[string]any {
			m := map[string]any{}
			maps.Copy(m, good)
			m["kid"] = "variant-" + k
			m[k] = val
			return m
		}
		return append([]any{
			map[string]any{"kty": "RSA", "kid": "rsa", "n": "AQAB", "e": "AQAB"},
			with("crv", "P-384"),
			with("use", "enc"),
			with("alg", "ES384"),
			with("x", "AAAA"),
			with("y", "!!!"),
		}, good)
	}})
	if _, err := v.Verify(t.Context(), iss.Token(t, authn.Identity{UserID: uuid.NewV7()})); err != nil {
		t.Errorf("a set with unusable keys beside a good one failed: %v", err)
	}
}

func TestVerifyFetchFailures(t *testing.T) {
	iss := authntest.NewIssuer(t)
	token := iss.Token(t, authn.Identity{UserID: uuid.NewV7()})
	tests := []struct {
		name string
		h    http.HandlerFunc
		want string
	}{
		{"error status", func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "down", http.StatusServiceUnavailable) }, "fetch jwks: 503 Service Unavailable"},
		{"not JSON", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "<html>") }, "decode jwks"},
		{"no keys", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{"keys":[]}`) }, "no ES256 signing keys"},
		{"too large", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"keys":[`+strings.Repeat(`{"kty":"oct"},`, 1<<17)+`{}]}`)
		}, "decode jwks"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := verifierFor(t, iss, tt.h).Verify(t.Context(), token)
			if !errors.Is(err, apperr.ErrUnauthorized) || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want ErrUnauthorized mentioning %q", err, tt.want)
			}
		})
	}
	t.Run("unreachable", func(t *testing.T) {
		cfg := iss.Config()
		cfg.JWKSURL = "http://127.0.0.1:1/jwks.json"
		if _, err := authn.NewVerifier(cfg, nil).Verify(t.Context(), token); !errors.Is(err, apperr.ErrUnauthorized) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("bad URL", func(t *testing.T) {
		cfg := iss.Config()
		cfg.JWKSURL = "http://bad host/"
		if _, err := authn.NewVerifier(cfg, nil).Verify(t.Context(), token); !errors.Is(err, apperr.ErrUnauthorized) {
			t.Errorf("err = %v", err)
		}
	})
}

func TestVerifyKeepsKeysWhenARefetchFails(t *testing.T) {
	iss := authntest.NewIssuer(t)
	proxy := &jwksProxy{iss: iss}
	v := verifierFor(t, iss, proxy)
	clk := newClock(v)
	user := authn.Identity{UserID: uuid.NewV7()}
	token := iss.Token(t, user)
	if _, err := v.Verify(t.Context(), token); err != nil {
		t.Fatal(err)
	}

	proxy.failing.Store(true)
	clk.advance(time.Minute)
	if _, err := v.Verify(t.Context(), iss.Token(t, user, authntest.WithKID("new"))); err == nil || !strings.Contains(err.Error(), "503") {
		t.Errorf("err = %v, want the fetch failure", err)
	}
	if _, err := v.Verify(t.Context(), token); err != nil {
		t.Errorf("a failed refetch dropped the cached keys: %v", err)
	}
}

func TestVerifyStopsWaitingWhenTheCallerLeaves(t *testing.T) {
	iss := authntest.NewIssuer(t)
	release := make(chan struct{})
	entered := make(chan struct{}, 1)
	v := verifierFor(t, iss, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		<-release
		(&jwksProxy{iss: iss}).ServeHTTP(w, r)
	}))
	token := iss.Token(t, authn.Identity{UserID: uuid.NewV7()})

	ctx, cancel := context.WithCancel(t.Context())
	errs := make(chan error, 1)
	go func() {
		_, err := v.Verify(ctx, token)
		errs <- err
	}()
	<-entered
	cancel()
	if err := <-errs; !errors.Is(err, apperr.ErrUnauthorized) || !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want ErrUnauthorized and context.Canceled", err)
	}

	// The fetch carried on without the caller, for the next one.
	close(release)
	if _, err := v.Verify(t.Context(), token); err != nil {
		t.Errorf("Verify after the fetch = %v", err)
	}
	select {
	case <-entered:
		t.Error("the JWKS was fetched twice")
	default:
	}
}
