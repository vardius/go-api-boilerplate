package app

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	userv1 "github.com/vardius/go-api-boilerplate/gen/user/v1"
	"github.com/vardius/go-api-boilerplate/internal/auth/keys"
	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/config"
	"github.com/vardius/go-api-boilerplate/internal/platform/connectx"
)

// MaxSessions bounds how many sessions Sessions lists.
const MaxSessions = 100

// maxLoginCodeBytes bounds a login code before it is sent on. Real codes
// are 43 bytes.
const maxLoginCodeBytes = 256

const badLoginCode = "The sign-in code is invalid, used or expired."

// SignIn redeems a login code with the user service and starts a session
// for the user it signs in: a grant of every grantable scope to the web
// app, from the browser userAgent names.
func (s *Service) SignIn(ctx context.Context, code, userAgentHeader string) (Tokens, error) {
	if code == "" {
		return Tokens{}, apperr.Invalid("code", "is required")
	}
	if len(code) > maxLoginCodeBytes {
		return Tokens{}, apperr.New(apperr.ErrUnauthorized, badLoginCode)
	}
	res, err := s.logins.RedeemLoginCode(ctx, &userv1.RedeemLoginCodeRequest{Code: code})
	if err != nil {
		return Tokens{}, redeemError(err)
	}
	user, err := uuid.Parse(res.GetUserId())
	if err != nil || user == uuid.Nil() {
		return Tokens{}, fmt.Errorf("redeem login code: the user service answered with user ID %q", res.GetUserId())
	}

	g, first, tokens, err := s.newGrant(user, authn.FirstPartyClient, GrantableScopes(), userAgent(userAgentHeader))
	if err != nil {
		return Tokens{}, err
	}
	if err := s.store.CreateGrant(ctx, g, first); err != nil {
		return Tokens{}, fmt.Errorf("start a session: %w", err)
	}
	return tokens, nil
}

// redeemError keeps what the user service's answer means for the caller: a
// code it refuses is the caller's 401, and a timeout is a timeout. Any
// other failure is the service's own and says nothing to the caller.
func redeemError(err error) error {
	err = connectx.FromConnect(err)
	switch {
	case errors.Is(err, apperr.ErrUnauthorized), errors.Is(err, apperr.ErrInvalid):
		return fmt.Errorf("redeem login code: %w: %v", apperr.New(apperr.ErrUnauthorized, badLoginCode), err)
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return fmt.Errorf("redeem login code: %w", err)
	default:
		return fmt.Errorf("redeem login code: %v", err)
	}
}

// errRefreshRejected is a refresh token that is unknown, expired, another
// client's, or reused.
var errRefreshRejected = errors.New("refresh token rejected")

// RefreshSession rotates a session's refresh token, issuing new tokens. A
// rejected token is apperr.ErrUnauthorized; one that had been rotated
// already also ends the session.
func (s *Service) RefreshSession(ctx context.Context, refresh string) (Tokens, error) {
	t, err := s.rotate(ctx, authn.FirstPartyClient, refresh, "")
	if errors.Is(err, errRefreshRejected) {
		return Tokens{}, fmt.Errorf("refresh session: %w: %w", apperr.New(apperr.ErrUnauthorized, "The session has ended; sign in again."), err)
	}
	return t, err
}

// rotate replaces refresh, which must have been issued to client, with a
// new token, and issues an access token for scope, or for the whole grant
// when scope is empty.
func (s *Service) rotate(ctx context.Context, client, refresh, scope string) (Tokens, error) {
	if refresh == "" {
		return Tokens{}, fmt.Errorf("no refresh token: %w", errRefreshRejected)
	}
	now := s.clock()
	h := store.HashToken(refresh)
	g, rotated, err := s.store.RefreshTokenGrant(ctx, h, now)
	switch {
	case errors.Is(err, apperr.ErrNotFound):
		return Tokens{}, fmt.Errorf("unknown or expired refresh token: %w", errRefreshRejected)
	case err != nil:
		return Tokens{}, fmt.Errorf("look up a refresh token: %w", err)
	case g.ClientID != client:
		return Tokens{}, fmt.Errorf("a refresh token of client %s presented by %s: %w", g.ClientID, client, errRefreshRejected)
	case rotated:
		return Tokens{}, s.revokeReused(ctx, g.ID)
	}
	scopes, err := requestedScopes(scope, g.Scopes)
	if err != nil {
		return Tokens{}, err
	}

	access, err := s.accessToken(g.UserID.String(), client, scopes, now)
	if err != nil {
		return Tokens{}, err
	}
	next := newSecret()
	nextToken := store.RefreshToken{Hash: store.HashToken(next), GrantID: g.ID, ExpiresAt: now.Add(RefreshTokenTTL)}
	switch err := s.store.RotateRefreshToken(ctx, h, nextToken, now); {
	case errors.Is(err, store.ErrUsed):
		return Tokens{}, s.revokeReused(ctx, g.ID)
	case errors.Is(err, apperr.ErrNotFound):
		return Tokens{}, fmt.Errorf("refresh token gone while rotating it: %w", errRefreshRejected)
	case err != nil:
		return Tokens{}, fmt.Errorf("rotate a refresh token: %w", err)
	}
	g.LastUsedAt, g.ExpiresAt = now, nextToken.ExpiresAt
	return Tokens{AccessToken: access, ExpiresIn: keys.AccessTokenTTL, Scopes: scopes, RefreshToken: config.Secret(next), Grant: g}, nil
}

// revokeReused ends a grant whose rotated refresh token came back.
func (s *Service) revokeReused(ctx context.Context, grant uuid.UUID) error {
	if err := s.store.RevokeGrant(ctx, grant); err != nil {
		return fmt.Errorf("revoke grant %s after its refresh token was reused: %w", grant, err)
	}
	return fmt.Errorf("refresh token reused; revoked grant %s: %w", grant, errRefreshRejected)
}

// SignOut ends the session refresh belongs to. Signing out without a
// session, or with one that has ended, does nothing.
func (s *Service) SignOut(ctx context.Context, refresh string) error {
	if refresh == "" {
		return nil
	}
	g, _, err := s.store.RefreshTokenGrant(ctx, store.HashToken(refresh), s.clock())
	switch {
	case errors.Is(err, apperr.ErrNotFound):
		return nil
	case err != nil:
		return fmt.Errorf("sign out: %w", err)
	case g.ClientID != authn.FirstPartyClient:
		return nil
	}
	if err := s.store.RevokeGrant(ctx, g.ID); err != nil {
		return fmt.Errorf("sign out: %w", err)
	}
	return nil
}

// Session is one of a user's sessions.
type Session struct {
	store.Grant
	// Current is set for the session the request comes from.
	Current bool
}

// Sessions lists user's sessions, the most recently used first, marking
// the one refresh, the caller's refresh token, belongs to.
func (s *Service) Sessions(ctx context.Context, user uuid.UUID, refresh string) ([]Session, error) {
	now := s.clock()
	grants, err := s.store.Grants(ctx, user, authn.FirstPartyClient, now, MaxSessions)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	current := uuid.Nil()
	if refresh != "" {
		g, _, err := s.store.RefreshTokenGrant(ctx, store.HashToken(refresh), now)
		switch {
		case err == nil && g.UserID == user:
			current = g.ID
		case err != nil && !errors.Is(err, apperr.ErrNotFound):
			return nil, fmt.Errorf("find the current session: %w", err)
		}
	}
	out := make([]Session, len(grants))
	for i, g := range grants {
		out[i] = Session{Grant: g, Current: g.ID == current}
	}
	return out, nil
}

// RevokeSession ends user's session id. Another user's session, or an
// OAuth grant, is apperr.ErrNotFound.
func (s *Service) RevokeSession(ctx context.Context, user, id uuid.UUID) error {
	g, err := s.store.Grant(ctx, id, s.clock())
	if err != nil && !errors.Is(err, apperr.ErrNotFound) {
		return fmt.Errorf("revoke session %s: %w", id, err)
	}
	if err != nil || g.UserID != user || g.ClientID != authn.FirstPartyClient {
		return fmt.Errorf("revoke session %s: %w", id, apperr.New(apperr.ErrNotFound, "No such session."))
	}
	if err := s.store.RevokeGrant(ctx, id); err != nil {
		return fmt.Errorf("revoke session %s: %w", id, err)
	}
	return nil
}
