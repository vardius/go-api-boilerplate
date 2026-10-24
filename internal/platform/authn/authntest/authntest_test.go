package authntest_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn/authntest"
)

func claims(t *testing.T, token string) (jwt.MapClaims, map[string]any) {
	t.Helper()
	c := jwt.MapClaims{}
	parsed, _, err := jwt.NewParser().ParseUnverified(token, c)
	if err != nil {
		t.Fatal(err)
	}
	return c, parsed.Header
}

func TestTokenDefaults(t *testing.T) {
	iss := authntest.NewIssuer(t)
	user := uuid.NewV7()
	c, h := claims(t, iss.Token(t, authn.Identity{UserID: user, Scopes: []string{"a", "b"}}))

	cfg := iss.Config()
	if c["iss"] != cfg.Issuer || c["sub"] != user.String() || c["client_id"] != authn.FirstPartyClient || c["scope"] != "a b" {
		t.Errorf("claims = %v", c)
	}
	if aud, _ := c.GetAudience(); len(aud) != 1 || aud[0] != cfg.Audience {
		t.Errorf("aud = %v", aud)
	}
	if exp, _ := c.GetExpirationTime(); time.Until(exp.Time) <= 14*time.Minute {
		t.Errorf("exp = %v, want 15 minutes from now", exp)
	}
	if h["typ"] != "at+jwt" || h["alg"] != "ES256" || h["kid"] == "" {
		t.Errorf("header = %v", h)
	}

	c, _ = claims(t, iss.Token(t, authn.Identity{ClientID: "auth-service"}))
	if c["sub"] != "auth-service" {
		t.Errorf("a client's own token has sub %v", c["sub"])
	}
}

func TestOptions(t *testing.T) {
	iss := authntest.NewIssuer(t)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	c, h := claims(t, iss.Token(t, authn.Identity{UserID: uuid.NewV7()},
		authntest.WithAudience("aud-x"), authntest.WithIssuer("iss-x"), authntest.WithKID("kid-x"),
		authntest.WithType("JWT"), authntest.WithKey(key), authntest.WithExpiry(-time.Hour),
		authntest.WithClaim("scope", nil), authntest.WithClaim("extra", "yes")))

	if aud, _ := c.GetAudience(); len(aud) != 1 || aud[0] != "aud-x" || c["iss"] != "iss-x" || c["extra"] != "yes" {
		t.Errorf("claims = %v", c)
	}
	if _, ok := c["scope"]; ok {
		t.Error("a nil WithClaim kept the claim")
	}
	if exp, _ := c.GetExpirationTime(); exp.After(time.Now()) {
		t.Errorf("exp = %v, want in the past", exp)
	}
	if h["kid"] != "kid-x" || h["typ"] != "JWT" {
		t.Errorf("header = %v", h)
	}
}

func TestRotateKeepsPublishingOldKeys(t *testing.T) {
	iss := authntest.NewIssuer(t)
	user := authn.Identity{UserID: uuid.NewV7()}
	before := iss.Token(t, user)
	iss.Rotate(t)
	after := iss.Token(t, user)

	_, h1 := claims(t, before)
	_, h2 := claims(t, after)
	if h1["kid"] == h2["kid"] {
		t.Fatal("Rotate kept the kid")
	}
	v := iss.Verifier()
	for _, token := range []string{before, after} {
		if _, err := v.Verify(t.Context(), token); err != nil {
			t.Error(err)
		}
	}
	if got := iss.JWKSRequests(); got != 1 {
		t.Errorf("JWKSRequests = %d, want 1", got)
	}
}

func TestOtherIssuersTokensFail(t *testing.T) {
	a, b := authntest.NewIssuer(t), authntest.NewIssuer(t)
	_, err := a.Verifier().Verify(t.Context(), b.Token(t, authn.Identity{UserID: uuid.NewV7()}))
	if !errors.Is(err, apperr.ErrUnauthorized) {
		t.Errorf("err = %v", err)
	}
}
