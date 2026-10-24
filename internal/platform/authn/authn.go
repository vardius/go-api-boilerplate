// Package authn verifies access tokens and says who a request is from.
//
// Access tokens are RFC 9068 JWTs that the auth service signs with ES256
// and publishes the keys for as a JWKS. A Verifier checks them locally, so
// no request costs a call to auth. Authenticate puts the caller's Identity
// in the request context, and Require, RequireUser and RequireFirstParty
// guard routes with it. The token itself is never kept: an Identity holds
// only IDs and scopes.
package authn

import (
	"context"
	"slices"
	"uuid"
)

// FirstPartyClient is the client ID of the web app, whose tokens come from
// a signed-in session rather than an OAuth grant.
const FirstPartyClient = "web"

// Identity is who a request is from.
type Identity struct {
	// Subject is the token's sub: the user ID for a user's token, the
	// client ID for a client's own token.
	Subject string
	// UserID is uuid.Nil() for a client's own token.
	UserID   uuid.UUID
	ClientID string
	Scopes   []string
}

// Has reports whether the token grants scope.
func (i Identity) Has(scope string) bool { return slices.Contains(i.Scopes, scope) }

// IsUser reports whether the token acts for a user.
func (i Identity) IsUser() bool { return i.UserID != uuid.Nil() }

// IsFirstParty reports whether the token was issued to the web app.
func (i Identity) IsFirstParty() bool { return i.ClientID == FirstPartyClient }

type identityKey struct{}

// NewContext returns ctx carrying id.
func NewContext(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, id)
}

// FromContext returns the Identity in ctx, and false for an anonymous
// request.
func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey{}).(Identity)
	return id, ok
}
