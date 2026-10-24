// Package store defines what the auth service keeps: OAuth clients,
// grants with their refresh tokens, authorization requests waiting for
// consent and authorization codes.
//
// Store is implemented by the memory, mysql and mongodb packages, and
// storetest is the contract each of them passes. Tokens and codes are never
// stored, only their SHA-256 (Hash), and every lookup by token goes through
// that hash's index. The operations that use something up, consuming an
// authorization request or code and rotating a refresh token, are atomic:
// of two concurrent uses exactly one succeeds.
//
// Times are kept to the millisecond. Something whose ExpiresAt has come is
// expired; lookups given now treat it as missing even before it is purged.
package store

import (
	"context"
	"crypto/sha256"
	"errors"
	"log/slog"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/run"
)

// ErrUsed means a single-use code, or a refresh token that has already
// been rotated, was presented again.
var ErrUsed = errors.New("store: already used")

// Missing lookups are apperr.ErrNotFound, and creating something whose ID
// or hash exists is apperr.ErrConflict.

// Hash is the SHA-256 of a token or code: what the store keeps instead of
// it.
type Hash [sha256.Size]byte

// HashToken returns the Hash of token.
func HashToken(token string) Hash { return sha256.Sum256([]byte(token)) }

// Client is a third-party app registered by a user.
type Client struct {
	ID           string
	OwnerID      uuid.UUID
	Name         string
	RedirectURIs []string
	Scopes       []string
	// SecretHash is the argon2id hash of the client secret, in PHC string
	// format.
	SecretHash string
	CreatedAt  time.Time
}

// Grant is a user's consent to a client, with a family of refresh tokens.
// A session of the web app is a grant to the first-party client.
type Grant struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	ClientID string
	Scopes   []string
	// UserAgent is the browser that started a session; empty for OAuth
	// grants.
	UserAgent  string
	CreatedAt  time.Time
	LastUsedAt time.Time
	// ExpiresAt is when the newest refresh token expires, and with it the
	// grant.
	ExpiresAt time.Time
}

// RefreshToken is one token of a grant's family.
type RefreshToken struct {
	Hash      Hash
	GrantID   uuid.UUID
	ExpiresAt time.Time
}

// AuthorizationRequest is a checked OAuth authorization request waiting for
// its user's decision.
type AuthorizationRequest struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	ClientID      string
	RedirectURI   string
	Scopes        []string
	State         string
	CodeChallenge string
	ExpiresAt     time.Time
}

// AuthorizationCode is an issued OAuth authorization code.
type AuthorizationCode struct {
	Hash          Hash
	ClientID      string
	UserID        uuid.UUID
	RedirectURI   string
	Scopes        []string
	CodeChallenge string
	ExpiresAt     time.Time
	// Used is set once the code has been redeemed or spent.
	Used bool
	// GrantID is the grant redeeming the code created, uuid.Nil() if none.
	GrantID uuid.UUID
}

// UsedCodeRetention is how long past its expiry a code is kept, so that a
// code replayed after it expired still revokes the grant it created.
const UsedCodeRetention = time.Hour

// Store keeps the auth service's state.
type Store interface {
	// CreateClient stores c.
	CreateClient(ctx context.Context, c Client) error
	// Client returns the client with id.
	Client(ctx context.Context, id string) (Client, error)
	// Clients returns owner's clients, oldest first, limit of them from
	// offset.
	Clients(ctx context.Context, owner uuid.UUID, limit, offset int) ([]Client, error)
	// DeleteClient deletes owner's client id with its grants, refresh
	// tokens, authorization requests and codes. Another owner's client is
	// apperr.ErrNotFound.
	DeleteClient(ctx context.Context, owner uuid.UUID, id string) error

	// CreateGrant stores g with its first refresh token.
	CreateGrant(ctx context.Context, g Grant, first RefreshToken) error
	// Grant returns the grant with id unless it has expired.
	Grant(ctx context.Context, id uuid.UUID, now time.Time) (Grant, error)
	// Grants returns up to limit of user's unexpired grants to client, the
	// most recently used first.
	Grants(ctx context.Context, user uuid.UUID, client string, now time.Time, limit int) ([]Grant, error)
	// RevokeGrant deletes grant id and its refresh tokens. Revoking a grant
	// that is gone does nothing.
	RevokeGrant(ctx context.Context, id uuid.UUID) error
	// RefreshTokenGrant returns the grant of the unexpired refresh token
	// with hash h, and whether that token has been rotated already.
	RefreshTokenGrant(ctx context.Context, h Hash, now time.Time) (g Grant, rotated bool, err error)
	// RotateRefreshToken replaces the token with hash h by next: it marks h
	// rotated, stores next and moves the grant's LastUsedAt to now and its
	// ExpiresAt to next's. A token rotated already is ErrUsed; an unknown or
	// expired one is apperr.ErrNotFound.
	RotateRefreshToken(ctx context.Context, h Hash, next RefreshToken, now time.Time) error

	// CreateAuthorizationRequest stores r.
	CreateAuthorizationRequest(ctx context.Context, r AuthorizationRequest) error
	// AuthorizationRequest returns user's unexpired request id.
	AuthorizationRequest(ctx context.Context, id, user uuid.UUID, now time.Time) (AuthorizationRequest, error)
	// TakeAuthorizationRequest deletes and returns user's unexpired
	// request id.
	TakeAuthorizationRequest(ctx context.Context, id, user uuid.UUID, now time.Time) (AuthorizationRequest, error)

	// CreateAuthorizationCode stores c, unused.
	CreateAuthorizationCode(ctx context.Context, c AuthorizationCode) error
	// AuthorizationCode returns the code with hash h, used or expired
	// ones included.
	AuthorizationCode(ctx context.Context, h Hash) (AuthorizationCode, error)
	// SpendAuthorizationCode marks the code used without creating a grant,
	// for an exchange that failed. A used code is ErrUsed.
	SpendAuthorizationCode(ctx context.Context, h Hash) error
	// RedeemAuthorizationCode marks the code used by g, creating g with its
	// first refresh token. A used code is ErrUsed, and nothing is created.
	RedeemAuthorizationCode(ctx context.Context, h Hash, g Grant, first RefreshToken) error
}

// Purger deletes what has expired: grants with their refresh tokens,
// rotated refresh tokens, authorization requests, and authorization codes
// UsedCodeRetention after they expired.
type Purger interface {
	Purge(ctx context.Context, now time.Time) error
}

// PurgeWorker purges p every interval, logging a failure and trying again
// at the next tick.
func PurgeWorker(p Purger, interval time.Duration, now func() time.Time, log *slog.Logger) run.Server {
	return run.Worker(func(ctx context.Context) error {
		tick := time.NewTicker(interval)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-tick.C:
				if err := p.Purge(ctx, now()); err != nil && ctx.Err() == nil {
					log.ErrorContext(ctx, "purge expired auth data", "err", err)
				}
			}
		}
	})
}
