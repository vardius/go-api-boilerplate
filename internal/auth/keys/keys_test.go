package keys_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/auth/keys"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
)

// rfc7517Key is the first EC key of RFC 7517 Appendix A.2. Its RFC 7638
// thumbprint, computed apart from this code with
//
//	printf '%s' '{"crv":"P-256","kty":"EC","x":"MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4","y":"4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM"}' |
//		openssl dgst -sha256 -binary | openssl base64 -A | tr '+/' '-_' | tr -d '='
//
// is rfc7517KID.
const (
	rfc7517D   = "870MB6gfuTJ4HtUnUvYMyJpr5eUZNP4Bk43bVdj3eAE"
	rfc7517X   = "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4"
	rfc7517Y   = "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM"
	rfc7517KID = "cn-I_WNMClehiVp51i_0VpOENW1upEerA8sEam5hn-s"
)

func rfc7517Key(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	d, err := base64.RawURLEncoding.DecodeString(rfc7517D)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ecdsa.ParseRawPrivateKey(elliptic.P256(), d)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func newKey(t *testing.T, curve elliptic.Curve) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func writePEM(t *testing.T, path, typ string, der []byte) string {
	t.Helper()
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func privatePEM(t *testing.T, dir string, key *ecdsa.PrivateKey) string {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return writePEM(t, filepath.Join(dir, "signing.pem"), "PRIVATE KEY", der)
}

func publicPEM(t *testing.T, path string, key *ecdsa.PublicKey) {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		t.Fatal(err)
	}
	writePEM(t, path, "PUBLIC KEY", der)
}

type jwk struct {
	Kty, Crv, X, Y, Kid, Use, Alg string
}

func jwks(t *testing.T, k *keys.Keys) []jwk {
	t.Helper()
	rec := httptest.NewRecorder()
	k.JWKS().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil))
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("JWKS = %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	var set struct{ Keys []jwk }
	if err := json.Unmarshal(rec.Body.Bytes(), &set); err != nil {
		t.Fatal(err)
	}
	return set.Keys
}

func discard() *slog.Logger { return slog.New(slog.DiscardHandler) }

func TestKIDIsTheRFC7638Thumbprint(t *testing.T) {
	path := privatePEM(t, t.TempDir(), rfc7517Key(t))
	k, err := keys.Load(keys.Config{SigningKeyFile: path}, false, discard())
	if err != nil {
		t.Fatal(err)
	}
	if k.KID() != rfc7517KID {
		t.Errorf("kid = %s, want %s", k.KID(), rfc7517KID)
	}
	want := []jwk{{Kty: "EC", Crv: "P-256", X: rfc7517X, Y: rfc7517Y, Kid: rfc7517KID, Use: "sig", Alg: "ES256"}}
	if got := jwks(t, k); !slices.Equal(got, want) {
		t.Errorf("jwks = %+v, want %+v", got, want)
	}
}

func TestLoadPublishesPreviousKeys(t *testing.T) {
	dir := t.TempDir()
	signing := newKey(t, elliptic.P256())
	keyFile := privatePEM(t, dir, signing)
	previousDir := filepath.Join(dir, "previous")
	if err := os.Mkdir(previousDir, 0o700); err != nil {
		t.Fatal(err)
	}
	publicPEM(t, filepath.Join(previousDir, "a.pem"), &rfc7517Key(t).PublicKey)
	publicPEM(t, filepath.Join(previousDir, "b.pem"), &newKey(t, elliptic.P256()).PublicKey)
	// The current key listed again, and a file that isn't *.pem, add nothing.
	publicPEM(t, filepath.Join(previousDir, "c.pem"), &signing.PublicKey)
	publicPEM(t, filepath.Join(previousDir, "README.txt"), &newKey(t, elliptic.P256()).PublicKey)

	k, err := keys.Load(keys.Config{SigningKeyFile: keyFile, PreviousKeysDir: previousDir}, false, discard())
	if err != nil {
		t.Fatal(err)
	}
	set := jwks(t, k)
	if len(set) != 3 {
		t.Fatalf("published %d keys, want the signing key and two previous ones: %+v", len(set), set)
	}
	if set[0].Kid != k.KID() || set[1].Kid != rfc7517KID {
		t.Errorf("kids = %s, %s: want the signing key first, then the previous keys in file order", set[0].Kid, set[1].Kid)
	}
}

func TestLoadDevelopmentKey(t *testing.T) {
	var logs bytes.Buffer
	k, err := keys.Load(keys.Config{}, true, slog.New(slog.NewTextHandler(&logs, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if len(jwks(t, k)) != 1 || k.KID() == "" {
		t.Error("the generated key isn't published")
	}
	if !strings.Contains(logs.String(), "level=WARN") {
		t.Errorf("no warning about the ephemeral key: %q", logs.String())
	}
	other, err := keys.Load(keys.Config{}, true, discard())
	if err != nil {
		t.Fatal(err)
	}
	if other.KID() == k.KID() {
		t.Error("two starts generated the same key")
	}
}

func TestLoadRefuses(t *testing.T) {
	dir := t.TempDir()
	good := privatePEM(t, dir, newKey(t, elliptic.P256()))

	p384, err := x509.MarshalPKCS8PrivateKey(newKey(t, elliptic.P384()))
	if err != nil {
		t.Fatal(err)
	}
	sec1, err := x509.MarshalECPrivateKey(newKey(t, elliptic.P256()))
	if err != nil {
		t.Fatal(err)
	}
	pkcs8 := func(name string) string {
		return writePEM(t, filepath.Join(dir, name), "PRIVATE KEY", []byte("not DER"))
	}
	notPEM := filepath.Join(dir, "garbage.pem")
	if err := os.WriteFile(notPEM, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	previous := func(name string, write func(path string)) string {
		d := filepath.Join(dir, name)
		if err := os.Mkdir(d, 0o700); err != nil {
			t.Fatal(err)
		}
		write(filepath.Join(d, "k.pem"))
		return d
	}
	privateInPrevious := previous("private", func(path string) {
		der, _ := x509.MarshalPKCS8PrivateKey(newKey(t, elliptic.P256()))
		writePEM(t, path, "PRIVATE KEY", der)
	})
	p384InPrevious := previous("p384", func(path string) { publicPEM(t, path, &newKey(t, elliptic.P384()).PublicKey) })
	badPublic := previous("bad", func(path string) { writePEM(t, path, "PUBLIC KEY", []byte("not DER")) })

	tests := []struct {
		name      string
		cfg       keys.Config
		ephemeral bool
	}{
		{"no key outside development", keys.Config{}, false},
		{"a missing file, even in development", keys.Config{SigningKeyFile: filepath.Join(dir, "missing.pem")}, true},
		{"not PEM", keys.Config{SigningKeyFile: notPEM}, false},
		{"SEC1 instead of PKCS#8", keys.Config{SigningKeyFile: writePEM(t, filepath.Join(dir, "sec1.pem"), "EC PRIVATE KEY", sec1)}, false},
		{"bad DER", keys.Config{SigningKeyFile: pkcs8("bad.pem")}, false},
		{"P-384", keys.Config{SigningKeyFile: writePEM(t, filepath.Join(dir, "p384.pem"), "PRIVATE KEY", p384)}, false},
		{"a missing previous keys directory", keys.Config{SigningKeyFile: good, PreviousKeysDir: filepath.Join(dir, "nowhere")}, false},
		{"a private key among the previous keys", keys.Config{SigningKeyFile: good, PreviousKeysDir: privateInPrevious}, false},
		{"a P-384 previous key", keys.Config{SigningKeyFile: good, PreviousKeysDir: p384InPrevious}, false},
		{"a previous key that doesn't parse", keys.Config{SigningKeyFile: good, PreviousKeysDir: badPublic}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := keys.Load(tt.cfg, tt.ephemeral, discard()); err == nil {
				t.Fatal("loaded")
			}
		})
	}
	if _, err := keys.New(newKey(t, elliptic.P384())); err == nil {
		t.Error("New accepted a P-384 signing key")
	}
	if _, err := keys.New(newKey(t, elliptic.P256()), &newKey(t, elliptic.P384()).PublicKey); err == nil {
		t.Error("New accepted a P-384 previous key")
	}
}

// issuer serves k's JWKS and returns a verifier for it, as another service
// would run it, with the clock at now.
func issuer(t *testing.T, k *keys.Keys, now time.Time) *authn.Verifier {
	t.Helper()
	srv := httptest.NewServer(k.JWKS())
	t.Cleanup(srv.Close)
	v := authn.NewVerifier(authn.VerifierConfig{JWKSURL: srv.URL, Issuer: "https://example.com/api/auth", Audience: "go-api-boilerplate"}, srv.Client())
	v.Now = func() time.Time { return now }
	return v
}

func newKeys(t *testing.T, signing *ecdsa.PrivateKey, previous ...*ecdsa.PublicKey) *keys.Keys {
	t.Helper()
	k, err := keys.New(signing, previous...)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

var issuedAt = time.Date(2026, 10, 24, 9, 30, 0, 0, time.UTC)

func TestSignedTokensVerify(t *testing.T) {
	k := newKeys(t, newKey(t, elliptic.P256()))
	signer := keys.NewSigner(k, "https://example.com/api/auth", "go-api-boilerplate")
	user := "0199c1b0-7a3e-7c2a-9d0b-8f1e2a3b4c5d"
	token, err := signer.Sign(keys.Claims{Subject: user, ClientID: "web", Scopes: []string{"user:read", "clients:write"}, IssuedAt: issuedAt})
	if err != nil {
		t.Fatal(err)
	}

	id, err := issuer(t, k, issuedAt.Add(time.Minute)).Verify(t.Context(), token)
	if err != nil {
		t.Fatal(err)
	}
	if id.Subject != user || id.UserID.String() != user || id.ClientID != "web" || !slices.Equal(id.Scopes, []string{"user:read", "clients:write"}) {
		t.Errorf("identity = %+v", id)
	}

	header, claims := decode(t, token)
	if header["typ"] != "at+jwt" || header["alg"] != "ES256" || header["kid"] != k.KID() {
		t.Errorf("header = %v", header)
	}
	wantClaims := []string{"aud", "client_id", "exp", "iat", "iss", "jti", "scope", "sub"}
	if got := slices.Sorted(maps.Keys(claims)); !slices.Equal(got, wantClaims) {
		t.Errorf("claims = %v, want %v", got, wantClaims)
	}
	if claims["exp"].(float64)-claims["iat"].(float64) != 900 {
		t.Errorf("lifetime = %vs, want 900s", claims["exp"].(float64)-claims["iat"].(float64))
	}
	if claims["iss"] != "https://example.com/api/auth" || claims["scope"] != "user:read clients:write" {
		t.Errorf("claims = %v", claims)
	}

	t.Run("expired", func(t *testing.T) {
		if _, err := issuer(t, k, issuedAt.Add(16*time.Minute)).Verify(t.Context(), token); !errors.Is(err, apperr.ErrUnauthorized) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("each token has its own jti", func(t *testing.T) {
		again, err := signer.Sign(keys.Claims{Subject: user, ClientID: "web", IssuedAt: issuedAt})
		if err != nil {
			t.Fatal(err)
		}
		if _, c := decode(t, again); c["jti"] == claims["jti"] {
			t.Error("jti repeated")
		}
	})
	t.Run("the service's own verifier", func(t *testing.T) {
		v := signer.Verifier()
		v.Now = func() time.Time { return issuedAt }
		if _, err := v.Verify(t.Context(), token); err != nil {
			t.Error(err)
		}
		other := keys.NewSigner(newKeys(t, newKey(t, elliptic.P256())), "https://example.com/api/auth", "go-api-boilerplate")
		forged, err := other.Sign(keys.Claims{Subject: user, ClientID: "web", IssuedAt: issuedAt})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := v.Verify(t.Context(), forged); err == nil {
			t.Error("a token from another key verified")
		}
	})
}

func TestSignRefuses(t *testing.T) {
	signer := keys.NewSigner(newKeys(t, newKey(t, elliptic.P256())), "https://example.com/api/auth", "go-api-boilerplate")
	tests := []struct {
		name   string
		claims keys.Claims
	}{
		{"the service scope", keys.Claims{Subject: "c", ClientID: "c", Scopes: []string{"user:read", keys.ServiceScope}, IssuedAt: issuedAt}},
		{"no subject", keys.Claims{ClientID: "c", IssuedAt: issuedAt}},
		{"no client", keys.Claims{Subject: "c", IssuedAt: issuedAt}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if token, err := signer.Sign(tt.claims); err == nil {
				t.Fatalf("signed %s", token)
			}
		})
	}
}

func TestRotation(t *testing.T) {
	old, current := newKey(t, elliptic.P256()), newKey(t, elliptic.P256())
	before := keys.NewSigner(newKeys(t, old), "https://example.com/api/auth", "go-api-boilerplate")
	token, err := before.Sign(keys.Claims{Subject: "client-1", ClientID: "client-1", IssuedAt: issuedAt})
	if err != nil {
		t.Fatal(err)
	}

	rotated := newKeys(t, current, &old.PublicKey)
	if _, err := issuer(t, rotated, issuedAt).Verify(t.Context(), token); err != nil {
		t.Errorf("a token signed with a previous key: %v", err)
	}
	removed := newKeys(t, current)
	if _, err := issuer(t, removed, issuedAt).Verify(t.Context(), token); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Errorf("a token signed with a removed key: err = %v", err)
	}
}

func TestServiceToken(t *testing.T) {
	k := newKeys(t, newKey(t, elliptic.P256()))
	now := issuedAt
	st := keys.NewServiceToken(keys.NewSigner(k, "https://example.com/api/auth", "go-api-boilerplate"), func() time.Time { return now })

	first, err := st.Token(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	id, err := issuer(t, k, now).Verify(t.Context(), first)
	if err != nil {
		t.Fatal(err)
	}
	if id.Subject != "auth" || id.ClientID != "auth" || id.IsUser() || !slices.Equal(id.Scopes, []string{"login-codes:redeem"}) {
		t.Errorf("identity = %+v", id)
	}
	if _, c := decode(t, first); c["exp"].(float64)-c["iat"].(float64) != 300 {
		t.Errorf("lifetime = %vs, want 300s", c["exp"].(float64)-c["iat"].(float64))
	}

	now = issuedAt.Add(4*time.Minute - time.Second)
	if again, _ := st.Token(t.Context()); again != first {
		t.Error("renewed with more than a minute left")
	}
	now = issuedAt.Add(4 * time.Minute)
	renewed, err := st.Token(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if renewed == first {
		t.Error("not renewed with a minute left")
	}
	if _, err := issuer(t, k, now.Add(4*time.Minute+30*time.Second)).Verify(t.Context(), renewed); err != nil {
		t.Errorf("the renewed token: %v", err)
	}
}

func decode(t *testing.T, token string) (header, claims map[string]any) {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts", len(parts))
	}
	for i, dst := range []*map[string]any{&header, &claims} {
		data, err := base64.RawURLEncoding.DecodeString(parts[i])
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, dst); err != nil {
			t.Fatal(err)
		}
	}
	return header, claims
}
