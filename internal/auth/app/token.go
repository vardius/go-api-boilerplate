package app

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/auth/keys"
	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
)

// The grant types the token endpoint takes.
const (
	GrantAuthorizationCode = "authorization_code"
	GrantClientCredentials = "client_credentials"
	GrantRefreshToken      = "refresh_token"
)

// tokenParams are the token request parameters this server reads, none of
// which may be repeated (RFC 6749 §3.2).
var tokenParams = []string{"grant_type", "code", "redirect_uri", "code_verifier", "scope", "refresh_token"}

// Token answers a token request from client c, whose credentials have been
// checked: RFC 6749 §4.1.3 with PKCE, §4.4.2 and §6. A request that fails
// is an *OAuthError; any other error is the server's.
func (s *Service) Token(ctx context.Context, c store.Client, form url.Values) (Tokens, error) {
	for _, p := range tokenParams {
		if len(form[p]) > 1 {
			return Tokens{}, oauthError(ErrorInvalidRequest, "The "+p+" parameter is repeated.")
		}
	}
	// A parameter without a value counts as omitted (RFC 6749 §3.1), as
	// form.Get has it.
	switch form.Get("grant_type") {
	case GrantAuthorizationCode:
		return s.exchangeCode(ctx, c, form)
	case GrantClientCredentials:
		return s.clientCredentials(c, form.Get("scope"))
	case GrantRefreshToken:
		return s.refreshToken(ctx, c, form)
	case "":
		return Tokens{}, oauthError(ErrorInvalidRequest, "The grant_type parameter is required.")
	default:
		return Tokens{}, oauthError(ErrorUnsupportedGrantType, "The grant_type must be authorization_code, client_credentials or refresh_token.")
	}
}

func (s *Service) clientCredentials(c store.Client, scope string) (Tokens, error) {
	scopes, err := requestedScopes(scope, c.Scopes)
	if err != nil {
		return Tokens{}, err
	}
	access, err := s.accessToken(c.ID, c.ID, scopes, s.clock())
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, ExpiresIn: keys.AccessTokenTTL, Scopes: scopes}, nil
}

func (s *Service) refreshToken(ctx context.Context, c store.Client, form url.Values) (Tokens, error) {
	refresh := form.Get("refresh_token")
	if refresh == "" {
		return Tokens{}, oauthError(ErrorInvalidRequest, "The refresh_token parameter is required.")
	}
	t, err := s.rotate(ctx, c.ID, refresh, form.Get("scope"))
	if errors.Is(err, errRefreshRejected) {
		return Tokens{}, oauthError(ErrorInvalidGrant, "The refresh token is invalid, expired, revoked or another client's.").because(err)
	}
	return t, err
}

var errBadCode = oauthError(ErrorInvalidGrant, "The authorization code is invalid, expired, used, or doesn't match the redirect_uri or code_verifier.")

// exchangeCode redeems an authorization code. The code must be unused and
// unexpired, and the request must come from the client it was issued to,
// with its redirect URI and the verifier of its challenge. A request that
// fails any of those spends the code, and a code presented again revokes
// the grant it created (RFC 6749 §4.1.2, RFC 9700 §4.2.1).
func (s *Service) exchangeCode(ctx context.Context, c store.Client, form url.Values) (Tokens, error) {
	code := form.Get("code")
	if code == "" {
		return Tokens{}, oauthError(ErrorInvalidRequest, "The code parameter is required.")
	}
	h := store.HashToken(code)
	ac, err := s.store.AuthorizationCode(ctx, h)
	switch {
	case errors.Is(err, apperr.ErrNotFound):
		return Tokens{}, badCode(errors.New("unknown code"))
	case err != nil:
		return Tokens{}, fmt.Errorf("look up an authorization code: %w", err)
	case ac.Used:
		return Tokens{}, s.replayedCode(ctx, ac)
	case !s.clock().Before(ac.ExpiresAt):
		return Tokens{}, badCode(fmt.Errorf("code of client %s expired", ac.ClientID))
	}

	var mismatch string
	verifier := form.Get("code_verifier")
	switch {
	case ac.ClientID != c.ID:
		mismatch = "issued to client " + ac.ClientID + ", presented by " + c.ID
	case form.Get("redirect_uri") != ac.RedirectURI:
		mismatch = "redirect_uri missing or different"
	case verifier == "":
		mismatch = "no code_verifier"
	case !validVerifier(verifier):
		mismatch = "malformed code_verifier"
	case subtle.ConstantTimeCompare([]byte(challenge(verifier)), []byte(ac.CodeChallenge)) != 1:
		mismatch = "wrong code_verifier"
	}
	if mismatch != "" {
		switch err := s.store.SpendAuthorizationCode(ctx, h); {
		case errors.Is(err, store.ErrUsed):
			return Tokens{}, s.replayed(ctx, h)
		case err != nil:
			return Tokens{}, fmt.Errorf("spend an authorization code: %w", err)
		}
		return Tokens{}, badCode(errors.New(mismatch))
	}

	g, first, tokens, err := s.newGrant(ac.UserID, ac.ClientID, ac.Scopes, "")
	if err != nil {
		return Tokens{}, err
	}
	switch err := s.store.RedeemAuthorizationCode(ctx, h, g, first); {
	case errors.Is(err, store.ErrUsed):
		return Tokens{}, s.replayed(ctx, h)
	case err != nil:
		return Tokens{}, fmt.Errorf("redeem an authorization code: %w", err)
	}
	return tokens, nil
}

func badCode(err error) *OAuthError {
	e := *errBadCode
	return e.because(err)
}

// replayed handles a code that another request used while this one was
// checking it.
func (s *Service) replayed(ctx context.Context, h store.Hash) error {
	ac, err := s.store.AuthorizationCode(ctx, h)
	if err != nil {
		return fmt.Errorf("look up a replayed authorization code: %w", err)
	}
	return s.replayedCode(ctx, ac)
}

// replayedCode revokes the grant a used code created, if any.
func (s *Service) replayedCode(ctx context.Context, ac store.AuthorizationCode) error {
	if ac.GrantID != uuid.Nil() {
		if err := s.store.RevokeGrant(ctx, ac.GrantID); err != nil {
			return fmt.Errorf("revoke grant %s of a replayed authorization code: %w", ac.GrantID, err)
		}
		return badCode(fmt.Errorf("code of client %s used again; revoked grant %s", ac.ClientID, ac.GrantID))
	}
	return badCode(fmt.Errorf("code of client %s used again", ac.ClientID))
}

// challenge returns the S256 code challenge of verifier, RFC 7636 §4.2.
func challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

const (
	unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
	base64URL  = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
)

// validVerifier reports whether v is a code verifier: 43 to 128
// unreserved characters, RFC 7636 §4.1.
func validVerifier(v string) bool {
	return len(v) >= 43 && len(v) <= 128 && only(v, unreserved)
}

// validChallenge reports whether c can be an S256 challenge: the 43
// characters of a SHA-256 in unpadded base64url.
func validChallenge(c string) bool {
	return len(c) == 43 && only(c, base64URL)
}

func only(s, alphabet string) bool {
	return !strings.ContainsFunc(s, func(r rune) bool { return !strings.ContainsRune(alphabet, r) })
}

// Revoke answers a revocation request from client c, RFC 7009: a refresh
// token issued to c ends its grant. Any other token is no error, so as not
// to tell a client what exists: an unknown or expired token, another
// client's, or an access token, which expires on its own in
// keys.AccessTokenTTL.
func (s *Service) Revoke(ctx context.Context, c store.Client, form url.Values) error {
	for _, p := range []string{"token", "token_type_hint"} {
		if len(form[p]) > 1 {
			return oauthError(ErrorInvalidRequest, "The "+p+" parameter is repeated.")
		}
	}
	token := form.Get("token")
	if token == "" {
		return oauthError(ErrorInvalidRequest, "The token parameter is required.")
	}
	// token_type_hint is only a hint (RFC 7009 §2.1); refresh tokens are
	// the only kind stored.
	g, _, err := s.store.RefreshTokenGrant(ctx, store.HashToken(token), s.clock())
	switch {
	case errors.Is(err, apperr.ErrNotFound):
		return nil
	case err != nil:
		return fmt.Errorf("revoke: %w", err)
	case g.ClientID != c.ID:
		return nil
	}
	if err := s.store.RevokeGrant(ctx, g.ID); err != nil {
		return fmt.Errorf("revoke grant %s: %w", g.ID, err)
	}
	return nil
}
