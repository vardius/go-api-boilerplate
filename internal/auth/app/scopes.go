package app

import (
	"slices"
	"strings"
)

// The scopes a user can grant. There is no scope that implies the others,
// and keys.ServiceScope, the one other scope tokens carry, is never among
// them.
const (
	ScopeUserRead     = "user:read"
	ScopeUserWrite    = "user:write"
	ScopeClientsRead  = "clients:read"
	ScopeClientsWrite = "clients:write"
)

// Scope is a scope and what it lets a client do, as the consent page says.
type Scope struct {
	Name        string
	Description string
}

var grantable = []Scope{
	{ScopeUserRead, "See your user ID and email address."},
	{ScopeUserWrite, "Change your email address and link a Google account."},
	{ScopeClientsRead, "See the OAuth apps you registered."},
	{ScopeClientsWrite, "Register and delete OAuth apps for you."},
}

// GrantableScopes returns the names of the scopes a user can grant, which
// a session also gets.
func GrantableScopes() []string {
	names := make([]string, len(grantable))
	for i, s := range grantable {
		names[i] = s.Name
	}
	return names
}

func describe(names []string) []Scope {
	out := make([]Scope, 0, len(names))
	for _, s := range grantable {
		if slices.Contains(names, s.Name) {
			out = append(out, s)
		}
	}
	return out
}

// canonicalScopes returns names without repeats, in the order of
// grantable, and false when one isn't grantable.
func canonicalScopes(names []string) ([]string, bool) {
	out := make([]string, 0, len(names))
	for _, s := range grantable {
		if slices.Contains(names, s.Name) {
			out = append(out, s.Name)
		}
	}
	for _, n := range names {
		if !slices.Contains(out, n) {
			return nil, false
		}
	}
	return out, true
}

// requestedScopes parses an OAuth scope parameter, space-separated as RFC
// 6749 §3.3 has it, and checks it against allowed. An empty one asks for
// all of allowed.
func requestedScopes(scope string, allowed []string) ([]string, error) {
	names := strings.Fields(scope)
	if len(names) == 0 {
		return slices.Clone(allowed), nil
	}
	scopes, ok := canonicalScopes(names)
	if !ok {
		return nil, oauthError(ErrorInvalidScope, "A requested scope does not exist.")
	}
	for _, s := range scopes {
		if !slices.Contains(allowed, s) {
			return nil, oauthError(ErrorInvalidScope, "The scope "+s+" is wider than the client may have.")
		}
	}
	return scopes, nil
}
