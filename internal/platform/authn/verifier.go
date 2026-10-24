package authn

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
)

// VerifierConfig says whose tokens to accept. Services nest it with a
// prefix, as in envPrefix:"AUTH_".
type VerifierConfig struct {
	JWKSURL  string `env:"JWKS_URL,required" desc:"URL of the auth service's JWKS, the keys access tokens are signed with."`
	Issuer   string `env:"ISSUER,required" desc:"The iss every access token must carry: the auth API's public URL."`
	Audience string `env:"AUDIENCE" envDefault:"go-api-boilerplate" desc:"The aud every access token must include."`
}

// TokenVerifier checks an access token and returns who it is for.
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (Identity, error)
}

const (
	leeway = 30 * time.Second
	// refetchInterval spaces out the fetches an unknown kid causes, so
	// tokens with made-up kids can't make the verifier hammer auth.
	refetchInterval = time.Minute
	// maxKeyAge bounds how long a key removed from the JWKS, such as a
	// compromised one, is still trusted.
	maxKeyAge    = time.Hour
	fetchTimeout = 10 * time.Second
	maxJWKSBytes = 1 << 20
)

// Verifier verifies access tokens against the keys in a JWKS. It is safe
// for concurrent use.
type Verifier struct {
	cfg    VerifierConfig
	client *http.Client
	// Now is the clock. Tests replace it before the first Verify.
	Now func() time.Time

	mu          sync.Mutex
	keys        map[string]*ecdsa.PublicKey // by kid; nil until a fetch succeeds
	fetchedAt   time.Time
	lastAttempt time.Time
	fetching    chan struct{} // closed when the fetch in flight ends
	fetchErr    error
}

var _ TokenVerifier = (*Verifier)(nil)

// NewVerifier returns a Verifier for cfg that fetches the JWKS with client,
// or with a client with a 10-second timeout when client is nil.
//
// The key set is fetched on first use and cached. A token whose kid isn't
// in it causes a refetch, at most once a minute, so rotated keys are
// picked up; the set is also refetched in the background once it is an hour
// old, so a key auth stops publishing stops being trusted.
func NewVerifier(cfg VerifierConfig, client *http.Client) *Verifier {
	if client == nil {
		client = &http.Client{Timeout: fetchTimeout}
	}
	return &Verifier{cfg: cfg, client: client, Now: time.Now}
}

type claims struct {
	jwt.RegisteredClaims
	ClientID string `json:"client_id"`
	Scope    string `json:"scope"`
}

// Verify checks token's ES256 signature, its at+jwt type, issuer, audience
// and expiry, with 30 seconds of leeway for clock skew. Every failure is an
// apperr.ErrUnauthorized; the reason is in the wrapped error, for logs.
func (v *Verifier) Verify(ctx context.Context, token string) (Identity, error) {
	id, err := v.verify(ctx, token)
	if err != nil {
		return Identity{}, fmt.Errorf("verify access token: %w: %w", apperr.ErrUnauthorized, err)
	}
	return id, nil
}

func (v *Verifier) verify(ctx context.Context, token string) (Identity, error) {
	var c claims
	_, err := jwt.ParseWithClaims(token, &c, func(t *jwt.Token) (any, error) {
		// RFC 9068 types access tokens so that an ID token, signed by the
		// same keys, can't be used as one.
		if typ, _ := t.Header["typ"].(string); !strings.EqualFold(typ, "at+jwt") && !strings.EqualFold(typ, "application/at+jwt") {
			return nil, fmt.Errorf("typ %q is not at+jwt", typ)
		}
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("no kid")
		}
		return v.key(ctx, kid)
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodES256.Alg()}),
		jwt.WithIssuer(v.cfg.Issuer),
		jwt.WithAudience(v.cfg.Audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(leeway),
		jwt.WithTimeFunc(v.Now),
	)
	if err != nil {
		return Identity{}, err
	}
	if c.Subject == "" || c.ClientID == "" {
		return Identity{}, errors.New("sub and client_id are required")
	}
	id := Identity{Subject: c.Subject, UserID: uuid.Nil(), ClientID: c.ClientID, Scopes: strings.Fields(c.Scope)}
	if c.Subject != c.ClientID {
		userID, err := uuid.Parse(c.Subject)
		if err != nil || userID == uuid.Nil() {
			return Identity{}, fmt.Errorf("sub %q is neither the client nor a user ID", c.Subject)
		}
		id.UserID = userID
	}
	return id, nil
}

// key returns the public key for kid, fetching the key set when it has none
// yet or doesn't know kid. Each call waits for at most one fetch, and only
// one runs at a time; it runs detached from ctx so that a caller giving up
// doesn't fail it for the callers waiting on it.
func (v *Verifier) key(ctx context.Context, kid string) (*ecdsa.PublicKey, error) {
	now := v.Now()
	v.mu.Lock()
	k, known := v.keys[kid]
	if v.fetching == nil {
		stale := v.keys != nil && now.Sub(v.fetchedAt) >= maxKeyAge
		missing := !known && (v.keys == nil || now.Sub(v.lastAttempt) >= refetchInterval)
		if stale || missing {
			v.fetching = make(chan struct{})
			v.lastAttempt = now
			go v.refresh(context.WithoutCancel(ctx), v.fetching)
		}
	}
	wait := v.fetching
	v.mu.Unlock()
	if known {
		return k, nil
	}
	if wait == nil {
		return nil, fmt.Errorf("unknown kid %q", kid)
	}

	select {
	case <-wait:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if k, ok := v.keys[kid]; ok {
		return k, nil
	}
	if v.fetchErr != nil {
		return nil, v.fetchErr
	}
	return nil, fmt.Errorf("unknown kid %q", kid)
}

func (v *Verifier) refresh(ctx context.Context, done chan struct{}) {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	keys, err := v.fetch(ctx)

	v.mu.Lock()
	defer v.mu.Unlock()
	if err == nil {
		v.keys, v.fetchedAt = keys, v.Now()
	}
	v.fetchErr = err
	v.fetching = nil
	close(done)
}

func (v *Verifier) fetch(ctx context.Context) (map[string]*ecdsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.cfg.JWKSURL, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	res, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch jwks: %s", res.Status)
	}
	var set struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, maxJWKSBytes)).Decode(&set); err != nil {
		return nil, fmt.Errorf("decode jwks: %w", err)
	}
	keys := make(map[string]*ecdsa.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if pub, ok := k.publicKey(); ok {
			keys[k.Kid] = pub
		}
	}
	// An empty set would fail every token; keep the keys already cached.
	if len(keys) == 0 {
		return nil, errors.New("jwks has no ES256 signing keys")
	}
	return keys, nil
}

type jwk struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

// publicKey returns the key if it is a P-256 signing key, skipping keys of
// other kinds so that one doesn't break the rest of the set.
func (k jwk) publicKey() (*ecdsa.PublicKey, bool) {
	if k.Kid == "" || k.Kty != "EC" || k.Crv != "P-256" || k.Use != "" && k.Use != "sig" || k.Alg != "" && k.Alg != "ES256" {
		return nil, false
	}
	x, errX := base64.RawURLEncoding.DecodeString(k.X)
	y, errY := base64.RawURLEncoding.DecodeString(k.Y)
	if errX != nil || errY != nil || len(x) != 32 || len(y) != 32 {
		return nil, false
	}
	pub, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), append(append([]byte{4}, x...), y...))
	return pub, err == nil
}
