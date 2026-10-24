// Package authntest issues access tokens for tests, the way the auth
// service does, from a JWKS server of its own.
package authntest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"

	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
)

// audience is the aud of the tokens an Issuer signs by default.
const audience = "go-api-boilerplate"

// Issuer signs ES256 access tokens and serves its public keys as a JWKS
// over an httptest server that closes when the test ends.
type Issuer struct {
	srv *httptest.Server

	mu       sync.Mutex
	keys     []signingKey // all published; the last one signs
	requests int
}

type signingKey struct {
	kid string
	key *ecdsa.PrivateKey
}

// NewIssuer starts an Issuer with one fresh key.
func NewIssuer(t testing.TB) *Issuer {
	t.Helper()
	iss := &Issuer{}
	iss.Rotate(t)
	iss.srv = httptest.NewServer(http.HandlerFunc(iss.serveJWKS))
	t.Cleanup(iss.srv.Close)
	return iss
}

// Config returns the VerifierConfig that accepts the Issuer's tokens.
func (i *Issuer) Config() authn.VerifierConfig {
	return authn.VerifierConfig{
		JWKSURL:  i.srv.URL + "/.well-known/jwks.json",
		Issuer:   i.srv.URL,
		Audience: audience,
	}
}

// Verifier returns a new Verifier for the Issuer's tokens.
func (i *Issuer) Verifier() *authn.Verifier {
	return authn.NewVerifier(i.Config(), i.srv.Client())
}

// Rotate starts signing with a new key, with a new kid. The keys before it
// stay published, as the auth service keeps serving previous keys.
func (i *Issuer) Rotate(t testing.TB) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.keys = append(i.keys, signingKey{kid: "test-" + uuid.NewV7().String(), key: key})
}

// JWKSRequests counts the requests the JWKS server has answered.
func (i *Issuer) JWKSRequests() int {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.requests
}

func (i *Issuer) serveJWKS(w http.ResponseWriter, _ *http.Request) {
	i.mu.Lock()
	i.requests++
	keys := make([]map[string]string, 0, len(i.keys))
	for _, k := range i.keys {
		// Bytes is 0x04 followed by the 32-byte X and Y coordinates.
		b, err := k.key.PublicKey.Bytes()
		if err != nil {
			i.mu.Unlock()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		keys = append(keys, map[string]string{
			"kty": "EC",
			"crv": "P-256",
			"use": "sig",
			"alg": "ES256",
			"kid": k.kid,
			"x":   base64.RawURLEncoding.EncodeToString(b[1:33]),
			"y":   base64.RawURLEncoding.EncodeToString(b[33:]),
		})
	}
	i.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"keys": keys})
}

// Option changes a token from its defaults.
type Option func(*tokenOptions)

type tokenOptions struct {
	expiresIn time.Duration
	audience  string
	issuer    string
	kid       string
	typ       string
	key       *ecdsa.PrivateKey
	claims    map[string]any
}

// WithExpiry makes the token expire d from now; a negative d gives an
// expired token. The default is 15 minutes.
func WithExpiry(d time.Duration) Option { return func(o *tokenOptions) { o.expiresIn = d } }

// WithAudience sets the token's aud.
func WithAudience(aud string) Option { return func(o *tokenOptions) { o.audience = aud } }

// WithIssuer sets the token's iss.
func WithIssuer(iss string) Option { return func(o *tokenOptions) { o.issuer = iss } }

// WithKID sets the kid header, which otherwise names the signing key.
func WithKID(kid string) Option { return func(o *tokenOptions) { o.kid = kid } }

// WithType sets the typ header, which is otherwise at+jwt.
func WithType(typ string) Option { return func(o *tokenOptions) { o.typ = typ } }

// WithKey signs the token with key instead of the Issuer's current key,
// keeping the current key's kid unless WithKID changes it too.
func WithKey(key *ecdsa.PrivateKey) Option { return func(o *tokenOptions) { o.key = key } }

// WithClaim sets claim name to value, overriding what Token would put
// there; a nil value leaves the claim out.
func WithClaim(name string, value any) Option {
	return func(o *tokenOptions) {
		if o.claims == nil {
			o.claims = map[string]any{}
		}
		o.claims[name] = value
	}
}

// Token signs an access token for id, as the auth service would. An empty
// ClientID is the web app's; an empty Subject is the user ID, or the client
// ID for a client acting for itself.
func (i *Issuer) Token(t testing.TB, id authn.Identity, opts ...Option) string {
	t.Helper()
	i.mu.Lock()
	current := i.keys[len(i.keys)-1]
	i.mu.Unlock()

	o := tokenOptions{expiresIn: 15 * time.Minute, audience: audience, issuer: i.srv.URL, kid: current.kid, typ: "at+jwt", key: current.key}
	for _, opt := range opts {
		opt(&o)
	}
	if id.ClientID == "" {
		id.ClientID = authn.FirstPartyClient
	}
	if id.Subject == "" {
		id.Subject = id.ClientID
		if id.IsUser() {
			id.Subject = id.UserID.String()
		}
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":       o.issuer,
		"sub":       id.Subject,
		"aud":       []string{o.audience},
		"exp":       now.Add(o.expiresIn).Unix(),
		"iat":       now.Unix(),
		"jti":       uuid.NewV7().String(),
		"client_id": id.ClientID,
		"scope":     strings.Join(id.Scopes, " "),
	}
	for name, value := range o.claims {
		if value == nil {
			delete(claims, name)
			continue
		}
		claims[name] = value
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["typ"] = o.typ
	token.Header["kid"] = o.kid
	signed, err := token.SignedString(o.key)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}
