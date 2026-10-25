package app_test

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"
	"uuid"

	"golang.org/x/oauth2"

	"github.com/vardius/go-api-boilerplate/internal/auth/app"
	"github.com/vardius/go-api-boilerplate/internal/auth/app/apptest"
	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
)

var errBoom = errors.New("boom")

// faulty is a store whose method named fail fails.
type faulty struct {
	store.Store
	fail string
}

func (f *faulty) broken(method string) bool { return f.fail == method }

func (f *faulty) CreateClient(ctx context.Context, c store.Client) error {
	if f.broken("CreateClient") {
		return errBoom
	}
	return f.Store.CreateClient(ctx, c)
}

func (f *faulty) Client(ctx context.Context, id string) (store.Client, error) {
	if f.broken("Client") {
		return store.Client{}, errBoom
	}
	return f.Store.Client(ctx, id)
}

func (f *faulty) Clients(ctx context.Context, owner uuid.UUID, after store.ClientKey, limit int) ([]store.Client, error) {
	if f.broken("Clients") {
		return nil, errBoom
	}
	return f.Store.Clients(ctx, owner, after, limit)
}

func (f *faulty) DeleteClient(ctx context.Context, owner uuid.UUID, id string) error {
	if f.broken("DeleteClient") {
		return errBoom
	}
	return f.Store.DeleteClient(ctx, owner, id)
}

func (f *faulty) CreateGrant(ctx context.Context, g store.Grant, first store.RefreshToken) error {
	if f.broken("CreateGrant") {
		return errBoom
	}
	return f.Store.CreateGrant(ctx, g, first)
}

func (f *faulty) Grant(ctx context.Context, id uuid.UUID, now time.Time) (store.Grant, error) {
	if f.broken("Grant") {
		return store.Grant{}, errBoom
	}
	return f.Store.Grant(ctx, id, now)
}

func (f *faulty) Grants(ctx context.Context, user uuid.UUID, client string, now time.Time, limit int) ([]store.Grant, error) {
	if f.broken("Grants") {
		return nil, errBoom
	}
	return f.Store.Grants(ctx, user, client, now, limit)
}

func (f *faulty) RevokeGrant(ctx context.Context, id uuid.UUID) error {
	if f.broken("RevokeGrant") {
		return errBoom
	}
	return f.Store.RevokeGrant(ctx, id)
}

func (f *faulty) RefreshTokenGrant(ctx context.Context, h store.Hash, now time.Time) (store.Grant, bool, error) {
	if f.broken("RefreshTokenGrant") {
		return store.Grant{}, false, errBoom
	}
	return f.Store.RefreshTokenGrant(ctx, h, now)
}

func (f *faulty) RotateRefreshToken(ctx context.Context, h store.Hash, next store.RefreshToken, now time.Time) error {
	if f.broken("RotateRefreshToken") {
		return errBoom
	}
	return f.Store.RotateRefreshToken(ctx, h, next, now)
}

func (f *faulty) CreateAuthorizationRequest(ctx context.Context, r store.AuthorizationRequest) error {
	if f.broken("CreateAuthorizationRequest") {
		return errBoom
	}
	return f.Store.CreateAuthorizationRequest(ctx, r)
}

func (f *faulty) AuthorizationRequest(ctx context.Context, id, user uuid.UUID, now time.Time) (store.AuthorizationRequest, error) {
	if f.broken("AuthorizationRequest") {
		return store.AuthorizationRequest{}, errBoom
	}
	return f.Store.AuthorizationRequest(ctx, id, user, now)
}

func (f *faulty) TakeAuthorizationRequest(ctx context.Context, id, user uuid.UUID, now time.Time) (store.AuthorizationRequest, error) {
	if f.broken("TakeAuthorizationRequest") {
		return store.AuthorizationRequest{}, errBoom
	}
	return f.Store.TakeAuthorizationRequest(ctx, id, user, now)
}

func (f *faulty) CreateAuthorizationCode(ctx context.Context, c store.AuthorizationCode) error {
	if f.broken("CreateAuthorizationCode") {
		return errBoom
	}
	return f.Store.CreateAuthorizationCode(ctx, c)
}

func (f *faulty) AuthorizationCode(ctx context.Context, h store.Hash) (store.AuthorizationCode, error) {
	if f.broken("AuthorizationCode") {
		return store.AuthorizationCode{}, errBoom
	}
	return f.Store.AuthorizationCode(ctx, h)
}

func (f *faulty) SpendAuthorizationCode(ctx context.Context, h store.Hash) error {
	if f.broken("SpendAuthorizationCode") {
		return errBoom
	}
	return f.Store.SpendAuthorizationCode(ctx, h)
}

func (f *faulty) RedeemAuthorizationCode(ctx context.Context, h store.Hash, g store.Grant, first store.RefreshToken) error {
	if f.broken("RedeemAuthorizationCode") {
		return errBoom
	}
	return f.Store.RedeemAuthorizationCode(ctx, h, g, first)
}

// A store failing is the server's error: it must never pass for a client
// error, such as a refresh token or code that doesn't exist.
func TestStoreFailuresAreServerErrors(t *testing.T) {
	type fixture struct {
		e        *apptest.Env
		s        *app.Service
		f        *faulty
		user     uuid.UUID
		session  app.Tokens
		client   store.Client
		verifier string
	}
	setup := func(t *testing.T) *fixture {
		e := apptest.New(t)
		f := &faulty{Store: e.Store}
		x := &fixture{e: e, s: app.New(f, e.Signer, e.Logins.Client, e.Clock.Now), f: f, verifier: oauth2.GenerateVerifier()}
		x.user, x.session = signIn(t, e)
		x.client, _ = newClient(t, e, x.user, "user:read")
		return x
	}
	query := func(x *fixture) url.Values {
		return authorizeQuery(x.client, oauth2.S256ChallengeFromVerifier(x.verifier))
	}
	request := func(t *testing.T, x *fixture) uuid.UUID {
		a := must(x.s.CheckAuthorization(t.Context(), query(x)))(t)
		return must(x.s.RequestConsent(t.Context(), x.user, a))(t)
	}

	tests := []struct {
		method string
		name   string
		run    func(t *testing.T, x *fixture, arm func()) error
	}{
		{"CreateGrant", "sign in", func(t *testing.T, x *fixture, arm func()) error {
			x.e.Logins.Add("c", uuid.NewV7(), "a@example.com")
			arm()
			_, err := x.s.SignIn(t.Context(), "c", "")
			return err
		}},
		{"RefreshTokenGrant", "refresh a session", func(t *testing.T, x *fixture, arm func()) error {
			arm()
			_, err := x.s.RefreshSession(t.Context(), x.session.RefreshToken.Reveal())
			return err
		}},
		{"RotateRefreshToken", "refresh a session", func(t *testing.T, x *fixture, arm func()) error {
			arm()
			_, err := x.s.RefreshSession(t.Context(), x.session.RefreshToken.Reveal())
			return err
		}},
		{"RevokeGrant", "reuse a refresh token", func(t *testing.T, x *fixture, arm func()) error {
			must(x.s.RefreshSession(t.Context(), x.session.RefreshToken.Reveal()))(t)
			arm()
			_, err := x.s.RefreshSession(t.Context(), x.session.RefreshToken.Reveal())
			return err
		}},
		{"RefreshTokenGrant", "sign out", func(t *testing.T, x *fixture, arm func()) error {
			arm()
			return x.s.SignOut(t.Context(), x.session.RefreshToken.Reveal())
		}},
		{"RevokeGrant", "sign out", func(t *testing.T, x *fixture, arm func()) error {
			arm()
			return x.s.SignOut(t.Context(), x.session.RefreshToken.Reveal())
		}},
		{"Grants", "list sessions", func(t *testing.T, x *fixture, arm func()) error {
			arm()
			_, err := x.s.Sessions(t.Context(), x.user, "")
			return err
		}},
		{"RefreshTokenGrant", "list sessions", func(t *testing.T, x *fixture, arm func()) error {
			arm()
			_, err := x.s.Sessions(t.Context(), x.user, x.session.RefreshToken.Reveal())
			return err
		}},
		{"Grant", "revoke a session", func(t *testing.T, x *fixture, arm func()) error {
			arm()
			return x.s.RevokeSession(t.Context(), x.user, x.session.Grant.ID)
		}},
		{"RevokeGrant", "revoke a session", func(t *testing.T, x *fixture, arm func()) error {
			arm()
			return x.s.RevokeSession(t.Context(), x.user, x.session.Grant.ID)
		}},
		{"CreateClient", "create a client", func(t *testing.T, x *fixture, arm func()) error {
			arm()
			_, _, err := x.s.CreateClient(t.Context(), x.user, app.NewClient{Name: "x", RedirectURIs: []string{redirectURI}, Scopes: []string{"user:read"}})
			return err
		}},
		{"Clients", "list clients", func(t *testing.T, x *fixture, arm func()) error {
			arm()
			_, err := x.s.Clients(t.Context(), x.user, 10, "")
			return err
		}},
		{"DeleteClient", "delete a client", func(t *testing.T, x *fixture, arm func()) error {
			arm()
			return x.s.DeleteClient(t.Context(), x.user, x.client.ID)
		}},
		{"Client", "authenticate a client", func(t *testing.T, x *fixture, arm func()) error {
			arm()
			_, err := x.s.AuthenticateClient(t.Context(), x.client.ID, "secret")
			return err
		}},
		{"Client", "check an authorization request", func(t *testing.T, x *fixture, arm func()) error {
			arm()
			_, err := x.s.CheckAuthorization(t.Context(), query(x))
			return err
		}},
		{"CreateAuthorizationRequest", "ask for consent", func(t *testing.T, x *fixture, arm func()) error {
			a := must(x.s.CheckAuthorization(t.Context(), query(x)))(t)
			arm()
			_, err := x.s.RequestConsent(t.Context(), x.user, a)
			return err
		}},
		{"AuthorizationRequest", "show consent", func(t *testing.T, x *fixture, arm func()) error {
			id := request(t, x)
			arm()
			_, err := x.s.Consent(t.Context(), x.user, id)
			return err
		}},
		{"Client", "show consent", func(t *testing.T, x *fixture, arm func()) error {
			id := request(t, x)
			arm()
			_, err := x.s.Consent(t.Context(), x.user, id)
			return err
		}},
		{"TakeAuthorizationRequest", "decide", func(t *testing.T, x *fixture, arm func()) error {
			id := request(t, x)
			arm()
			_, err := x.s.Decide(t.Context(), x.user, id, true)
			return err
		}},
		{"Client", "decide", func(t *testing.T, x *fixture, arm func()) error {
			id := request(t, x)
			arm()
			_, err := x.s.Decide(t.Context(), x.user, id, true)
			return err
		}},
		{"CreateAuthorizationCode", "decide", func(t *testing.T, x *fixture, arm func()) error {
			id := request(t, x)
			arm()
			_, err := x.s.Decide(t.Context(), x.user, id, true)
			return err
		}},
		{"AuthorizationCode", "exchange a code", func(t *testing.T, x *fixture, arm func()) error {
			code := authorize(t, x.e, x.user, query(x))
			arm()
			_, err := x.s.Token(t.Context(), x.client, exchange(code, x.verifier))
			return err
		}},
		{"RedeemAuthorizationCode", "exchange a code", func(t *testing.T, x *fixture, arm func()) error {
			code := authorize(t, x.e, x.user, query(x))
			arm()
			_, err := x.s.Token(t.Context(), x.client, exchange(code, x.verifier))
			return err
		}},
		{"SpendAuthorizationCode", "exchange a code with a wrong verifier", func(t *testing.T, x *fixture, arm func()) error {
			code := authorize(t, x.e, x.user, query(x))
			arm()
			_, err := x.s.Token(t.Context(), x.client, exchange(code, oauth2.GenerateVerifier()))
			return err
		}},
		{"RevokeGrant", "replay a code", func(t *testing.T, x *fixture, arm func()) error {
			code := authorize(t, x.e, x.user, query(x))
			must(x.s.Token(t.Context(), x.client, exchange(code, x.verifier)))(t)
			arm()
			_, err := x.s.Token(t.Context(), x.client, exchange(code, x.verifier))
			return err
		}},
		{"RefreshTokenGrant", "refresh an OAuth grant", func(t *testing.T, x *fixture, arm func()) error {
			tokens := authorizationCodeTokens(t, x.e, x.user, x.client)
			arm()
			_, err := x.s.Token(t.Context(), x.client, refreshForm(tokens.RefreshToken.Reveal(), ""))
			return err
		}},
		{"RefreshTokenGrant", "revoke", func(t *testing.T, x *fixture, arm func()) error {
			arm()
			return x.s.Revoke(t.Context(), x.client, url.Values{"token": {"t"}})
		}},
		{"RevokeGrant", "revoke", func(t *testing.T, x *fixture, arm func()) error {
			tokens := authorizationCodeTokens(t, x.e, x.user, x.client)
			arm()
			return x.s.Revoke(t.Context(), x.client, url.Values{"token": {tokens.RefreshToken.Reveal()}})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.method+" in "+tt.name, func(t *testing.T) {
			x := setup(t)
			err := tt.run(t, x, func() { x.f.fail = tt.method })
			if !errors.Is(err, errBoom) {
				t.Fatalf("err = %v, want the store's", err)
			}
			if _, ok := errors.AsType[*app.OAuthError](err); ok {
				t.Fatalf("a store failure passed for an OAuth error: %v", err)
			}
			if _, ok := errors.AsType[*app.AuthorizeError](err); ok {
				t.Fatalf("a store failure passed for an authorization error: %v", err)
			}
			for _, kind := range []error{apperr.ErrInvalid, apperr.ErrUnauthorized, apperr.ErrNotFound} {
				if errors.Is(err, kind) {
					t.Fatalf("a store failure is %v: %v", kind, err)
				}
			}
		})
	}
}
