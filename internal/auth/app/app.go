// Package app holds the auth service's use cases: sessions for the web app,
// OAuth 2 clients, and the authorization server that issues their tokens.
//
// A session and an OAuth grant are one thing, a store.Grant: a user's
// consent to a client, with a family of refresh tokens that rotate on every
// use. Presenting a token that has been rotated already means it leaked, so
// the whole grant is revoked. Access tokens are short-lived JWTs that keys
// signs; nothing about them is stored.
//
// The OAuth 2 server follows RFC 6749 as profiled by RFC 9700:
// authorization_code with PKCE S256 only, client_credentials and
// refresh_token grants, exact redirect URI matching, single-use codes, and
// RFC 7009 revocation. Methods take what the user or client sent and return
// apperr kinds or, at the OAuth endpoints, *OAuthError and *AuthorizeError.
// Token material never appears in an error's text.
package app

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/vardius/go-api-boilerplate/gen/user/v1/userv1connect"
	"github.com/vardius/go-api-boilerplate/internal/auth/keys"
	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/config"
)

// Lifetimes.
const (
	// RefreshTokenTTL is how long a refresh token lasts. Each rotation
	// issues a new one, so a grant used at least this often never expires.
	RefreshTokenTTL         = 30 * 24 * time.Hour
	AuthorizationRequestTTL = 10 * time.Minute
	AuthorizationCodeTTL    = time.Minute
)

// Service runs the use cases.
type Service struct {
	store  store.Store
	signer *keys.Signer
	logins userv1connect.LoginServiceClient
	now    func() time.Time
}

// New returns a Service keeping its state in st, signing tokens with
// signer, redeeming login codes with logins and reading the time from now.
func New(st store.Store, signer *keys.Signer, logins userv1connect.LoginServiceClient, now func() time.Time) *Service {
	return &Service{store: st, signer: signer, logins: logins, now: now}
}

// clock returns the time to the millisecond, which every store keeps.
func (s *Service) clock() time.Time { return s.now().UTC().Truncate(time.Millisecond) }

// Tokens is what a sign-in, a refresh or a token request issues.
type Tokens struct {
	AccessToken config.Secret
	ExpiresIn   time.Duration
	Scopes      []string
	// RefreshToken and Grant are empty for a client acting for itself.
	RefreshToken config.Secret
	Grant        store.Grant
}

// newSecret returns 32 random bytes, base64url-encoded: a refresh token,
// an authorization code or a client secret.
func newSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b) // Read never fails; it crashes the program instead.
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *Service) accessToken(subject, client string, scopes []string, now time.Time) (config.Secret, error) {
	token, err := s.signer.Sign(keys.Claims{Subject: subject, ClientID: client, Scopes: scopes, IssuedAt: now})
	if err != nil {
		return "", fmt.Errorf("sign an access token for %s: %w", client, err)
	}
	return config.Secret(token), nil
}

// newGrant returns a grant of scopes to client for user, with its first
// refresh token and the tokens to hand out once it is stored.
func (s *Service) newGrant(user uuid.UUID, client string, scopes []string, userAgent string) (store.Grant, store.RefreshToken, Tokens, error) {
	now := s.clock()
	g := store.Grant{
		ID:         uuid.NewV7(),
		UserID:     user,
		ClientID:   client,
		Scopes:     scopes,
		UserAgent:  userAgent,
		CreatedAt:  now,
		LastUsedAt: now,
		ExpiresAt:  now.Add(RefreshTokenTTL),
	}
	access, err := s.accessToken(user.String(), client, scopes, now)
	if err != nil {
		return store.Grant{}, store.RefreshToken{}, Tokens{}, err
	}
	refresh := newSecret()
	first := store.RefreshToken{Hash: store.HashToken(refresh), GrantID: g.ID, ExpiresAt: g.ExpiresAt}
	return g, first, Tokens{
		AccessToken:  access,
		ExpiresIn:    keys.AccessTokenTTL,
		Scopes:       scopes,
		RefreshToken: config.Secret(refresh),
		Grant:        g,
	}, nil
}

// maxUserAgentBytes bounds what a session keeps of its User-Agent.
const maxUserAgentBytes = 512

func userAgent(ua string) string {
	ua = strings.ToValidUTF8(ua, "�")
	if len(ua) <= maxUserAgentBytes {
		return ua
	}
	cut := maxUserAgentBytes
	for !utf8.RuneStart(ua[cut]) {
		cut--
	}
	return ua[:cut]
}

// validClientID reports whether id has the form of the IDs CreateClient
// gives, so that a malformed one never reaches the store.
func validClientID(id string) bool {
	parsed, err := uuid.Parse(id)
	return err == nil && parsed.String() == id
}
