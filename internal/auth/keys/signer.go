package keys

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"

	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
)

// AccessTokenTTL is how long an access token is valid.
const AccessTokenTTL = 15 * time.Minute

// ServiceScope lets a token redeem login codes with the user service. Only
// the ServiceToken carries it.
const ServiceScope = "login-codes:redeem"

// ServiceClientID is the sub and client_id of the service token.
const ServiceClientID = "auth"

const (
	serviceTokenTTL = 5 * time.Minute
	// serviceTokenRenewal is how long before expiry the service token is
	// replaced, which covers clock skew and a call in flight.
	serviceTokenRenewal = time.Minute
)

// Signer signs access tokens with the signing key of Keys.
type Signer struct {
	keys     *Keys
	issuer   string
	audience string
}

// NewSigner returns a Signer whose tokens carry issuer as iss and audience
// as aud.
func NewSigner(keys *Keys, issuer, audience string) *Signer {
	return &Signer{keys: keys, issuer: issuer, audience: audience}
}

// Issuer returns the iss of the tokens s signs.
func (s *Signer) Issuer() string { return s.issuer }

// Claims is what an access token says.
type Claims struct {
	// Subject is the user ID for a user's token and the client ID for a
	// client's own.
	Subject  string
	ClientID string
	Scopes   []string
	IssuedAt time.Time
}

// Sign returns an RFC 9068 access token for c that expires AccessTokenTTL
// after c.IssuedAt. It refuses ServiceScope.
func (s *Signer) Sign(c Claims) (string, error) {
	if slices.Contains(c.Scopes, ServiceScope) {
		return "", errors.New("keys: " + ServiceScope + " is only for the service token")
	}
	return s.sign(c, AccessTokenTTL)
}

func (s *Signer) sign(c Claims, ttl time.Duration) (string, error) {
	if c.Subject == "" || c.ClientID == "" {
		return "", errors.New("keys: a token needs a subject and a client")
	}
	claims := struct {
		jwt.RegisteredClaims
		ClientID string `json:"client_id"`
		Scope    string `json:"scope"`
	}{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   c.Subject,
			Audience:  jwt.ClaimStrings{s.audience},
			ExpiresAt: jwt.NewNumericDate(c.IssuedAt.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(c.IssuedAt),
			ID:        uuid.New().String(),
		},
		ClientID: c.ClientID,
		Scope:    strings.Join(c.Scopes, " "),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["typ"] = "at+jwt"
	token.Header["kid"] = s.keys.kid
	signed, err := token.SignedString(s.keys.signing)
	if err != nil {
		return "", fmt.Errorf("keys: sign an access token: %w", err)
	}
	return signed, nil
}

// Verifier returns an authn.Verifier for the tokens s signs that reads the
// keys from memory, so the service checks its own tokens by the same rules
// as every other service.
func (s *Signer) Verifier() *authn.Verifier {
	cfg := authn.VerifierConfig{
		// Never dialled: the transport answers every request itself.
		JWKSURL:  "http://jwks.invalid/.well-known/jwks.json",
		Issuer:   s.issuer,
		Audience: s.audience,
	}
	return authn.NewVerifier(cfg, &http.Client{Transport: jwksTransport{jwks: s.keys.jwks}})
}

// ServiceToken hands out the token the service calls the user service
// with, signing a new one shortly before the last expires. It is safe for
// concurrent use.
type ServiceToken struct {
	signer *Signer
	now    func() time.Time

	mu      sync.Mutex
	token   string
	expires time.Time
}

// NewServiceToken returns a ServiceToken signed by signer, reading the time
// from now.
func NewServiceToken(signer *Signer, now func() time.Time) *ServiceToken {
	return &ServiceToken{signer: signer, now: now}
}

// Token returns a token with ServiceScope whose sub and client_id are
// ServiceClientID, valid for at least another minute. It fits
// connectx.Bearer.
func (t *ServiceToken) Token(context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	if t.token != "" && now.Before(t.expires.Add(-serviceTokenRenewal)) {
		return t.token, nil
	}
	token, err := t.signer.sign(Claims{
		Subject:  ServiceClientID,
		ClientID: ServiceClientID,
		Scopes:   []string{ServiceScope},
		IssuedAt: now,
	}, serviceTokenTTL)
	if err != nil {
		return "", err
	}
	t.token, t.expires = token, now.Add(serviceTokenTTL)
	return token, nil
}
