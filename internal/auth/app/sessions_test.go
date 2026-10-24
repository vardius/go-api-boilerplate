package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
	"uuid"

	"connectrpc.com/connect/v2"

	"github.com/vardius/go-api-boilerplate/internal/auth/app"
	"github.com/vardius/go-api-boilerplate/internal/auth/app/apptest"
	"github.com/vardius/go-api-boilerplate/internal/auth/keys"
	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
)

func TestSignInStartsAWebSession(t *testing.T) {
	e := apptest.New(t)
	user := uuid.NewV7()
	e.Logins.Add("the-code", user, "ada@example.com")

	tokens := must(e.Service.SignIn(t.Context(), "the-code", "Mozilla/5.0"))(t)

	id := verify(t, e, tokens.AccessToken.Reveal())
	if id.UserID != user || !id.IsFirstParty() || strings.Join(id.Scopes, " ") != "user:read user:write clients:read clients:write" {
		t.Fatalf("access token for %+v", id)
	}
	if tokens.ExpiresIn != keys.AccessTokenTTL || tokens.RefreshToken == "" {
		t.Fatalf("tokens = %+v", tokens)
	}
	g := must(e.Store.Grant(t.Context(), tokens.Grant.ID, e.Clock.Now()))(t)
	if g.UserID != user || g.ClientID != authn.FirstPartyClient || g.UserAgent != "Mozilla/5.0" ||
		!g.ExpiresAt.Equal(e.Clock.Now().Add(app.RefreshTokenTTL)) {
		t.Fatalf("grant = %+v", g)
	}
	if got, _, err := e.Store.RefreshTokenGrant(t.Context(), store.HashToken(tokens.RefreshToken.Reveal()), e.Clock.Now()); err != nil || got.ID != g.ID {
		t.Fatalf("the refresh token belongs to %v (%v), want %v", got.ID, err, g.ID)
	}
	if e.Logins.Calls() != 1 {
		t.Fatalf("%d calls passed the service token check, want 1", e.Logins.Calls())
	}

	t.Run("a code works once", func(t *testing.T) {
		_, err := e.Service.SignIn(t.Context(), "the-code", "")
		isKind(t, err, apperr.ErrUnauthorized)
		if apperr.Message(err) == "" {
			t.Fatalf("no message for the client: %v", err)
		}
		noSecret(t, err, "the-code")
	})
}

func TestSignInRefusals(t *testing.T) {
	e := apptest.New(t)
	tests := []struct {
		name string
		code string
		fail error
		kind error
	}{
		{"no code", "", nil, apperr.ErrInvalid},
		{"a code too long to be one", strings.Repeat("a", 257), nil, apperr.ErrUnauthorized},
		{"an unknown code", "unknown", nil, apperr.ErrUnauthorized},
		{"a code the user service finds invalid", "x", connect.NewError(connect.CodeInvalidArgument, "bad code"), apperr.ErrUnauthorized},
		{"a user service timing out", "x", connect.NewError(connect.CodeDeadlineExceeded, "slow"), context.DeadlineExceeded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e.Logins.FailWith(tt.fail)
			_, err := e.Service.SignIn(t.Context(), tt.code, "")
			isKind(t, err, tt.kind)
		})
	}

	t.Run("a user service failing is the server's error", func(t *testing.T) {
		e.Logins.FailWith(connect.NewError(connect.CodeUnavailable, "down"))
		_, err := e.Service.SignIn(t.Context(), "x", "")
		for _, kind := range []error{apperr.ErrUnauthorized, apperr.ErrInvalid, apperr.ErrNotFound, context.DeadlineExceeded} {
			if errors.Is(err, kind) {
				t.Fatalf("err = %v is %v, want no kind", err, kind)
			}
		}
	})

	t.Run("a user service answering with no user ID", func(t *testing.T) {
		e.Logins.FailWith(nil)
		e.Logins.AddRaw("raw", "not-a-uuid")
		if _, err := e.Service.SignIn(t.Context(), "raw", ""); err == nil || errors.Is(err, apperr.ErrUnauthorized) {
			t.Fatalf("err = %v, want a server error", err)
		}
	})
}

func TestSignInKeepsAValidUserAgentOfBoundedSize(t *testing.T) {
	e := apptest.New(t)
	long := strings.Repeat("a", 511) + "é" + strings.Repeat("b", 100)
	for ua, want := range map[string]string{
		long:          strings.Repeat("a", 511),
		"bad \xff ua": "bad � ua",
	} {
		user := uuid.NewV7()
		e.Logins.Add("c", user, "a@example.com")
		tokens := must(e.Service.SignIn(t.Context(), "c", ua))(t)
		if got := tokens.Grant.UserAgent; got != want || !utf8.ValidString(got) {
			t.Errorf("user agent kept as %q, want %q", got, want)
		}
	}
}

func TestRefreshSessionRotates(t *testing.T) {
	e := apptest.New(t)
	user, first := signIn(t, e)
	e.Clock.Advance(time.Hour)

	second := must(e.Service.RefreshSession(t.Context(), first.RefreshToken.Reveal()))(t)
	if second.RefreshToken == first.RefreshToken || second.Grant.ID != first.Grant.ID {
		t.Fatalf("refresh gave %+v", second)
	}
	if id := verify(t, e, second.AccessToken.Reveal()); id.UserID != user || !id.IsFirstParty() || len(id.Scopes) != 4 {
		t.Fatalf("refreshed token for %+v", id)
	}
	g := must(e.Store.Grant(t.Context(), first.Grant.ID, e.Clock.Now()))(t)
	if !g.LastUsedAt.Equal(e.Clock.Now()) || !g.ExpiresAt.Equal(e.Clock.Now().Add(app.RefreshTokenTTL)) {
		t.Fatalf("grant after refresh = %+v", g)
	}

	t.Run("reusing the rotated token ends the session", func(t *testing.T) {
		_, err := e.Service.RefreshSession(t.Context(), first.RefreshToken.Reveal())
		isKind(t, err, apperr.ErrUnauthorized)
		noSecret(t, err, first.RefreshToken.Reveal())
		_, err = e.Service.RefreshSession(t.Context(), second.RefreshToken.Reveal())
		isKind(t, err, apperr.ErrUnauthorized)
		_, err = e.Store.Grant(t.Context(), first.Grant.ID, e.Clock.Now())
		isKind(t, err, apperr.ErrNotFound)
	})
}

func TestRefreshSessionRefusals(t *testing.T) {
	e := apptest.New(t)
	_, session := signIn(t, e)

	for name, token := range map[string]string{"no token": "", "an unknown token": "nope"} {
		t.Run(name, func(t *testing.T) {
			_, err := e.Service.RefreshSession(t.Context(), token)
			isKind(t, err, apperr.ErrUnauthorized)
		})
	}

	t.Run("an OAuth client's token, which stays valid", func(t *testing.T) {
		c, _ := newClient(t, e, uuid.NewV7(), "user:read")
		oauth := authorizationCodeTokens(t, e, uuid.NewV7(), c)
		_, err := e.Service.RefreshSession(t.Context(), oauth.RefreshToken.Reveal())
		isKind(t, err, apperr.ErrUnauthorized)
		must(e.Store.Grant(t.Context(), oauth.Grant.ID, e.Clock.Now()))(t)
	})

	t.Run("an expired token", func(t *testing.T) {
		e.Clock.Advance(app.RefreshTokenTTL)
		_, err := e.Service.RefreshSession(t.Context(), session.RefreshToken.Reveal())
		isKind(t, err, apperr.ErrUnauthorized)
	})
}

func TestSignOut(t *testing.T) {
	e := apptest.New(t)
	_, session := signIn(t, e)
	c, _ := newClient(t, e, uuid.NewV7(), "user:read")
	oauth := authorizationCodeTokens(t, e, uuid.NewV7(), c)

	for _, token := range []string{"", "unknown", oauth.RefreshToken.Reveal(), session.RefreshToken.Reveal(), session.RefreshToken.Reveal()} {
		if err := e.Service.SignOut(t.Context(), token); err != nil {
			t.Fatal(err)
		}
	}
	_, err := e.Store.Grant(t.Context(), session.Grant.ID, e.Clock.Now())
	isKind(t, err, apperr.ErrNotFound)
	if _, err := e.Store.Grant(t.Context(), oauth.Grant.ID, e.Clock.Now()); err != nil {
		t.Fatalf("signing out with an OAuth client's token revoked it: %v", err)
	}
}

func TestSessions(t *testing.T) {
	e := apptest.New(t)
	user := uuid.NewV7()
	var sessions []app.Tokens
	for range 3 {
		e.Logins.Add("c", user, "a@example.com")
		sessions = append(sessions, must(e.Service.SignIn(t.Context(), "c", ""))(t))
		e.Clock.Advance(time.Second)
	}
	signIn(t, e) // someone else
	c, _ := newClient(t, e, user, "user:read")
	authorizationCodeTokens(t, e, user, c)

	list := must(e.Service.Sessions(t.Context(), user, sessions[1].RefreshToken.Reveal()))(t)
	if len(list) != 3 {
		t.Fatalf("listed %d sessions, want 3", len(list))
	}
	for i, s := range list {
		want := sessions[2-i]
		if s.ID != want.Grant.ID || s.Current != (i == 1) {
			t.Errorf("session %d = %+v, want %v current=%v", i, s, want.Grant.ID, i == 1)
		}
	}

	t.Run("with someone else's or no refresh token none is current", func(t *testing.T) {
		_, other := signIn(t, e)
		for _, token := range []string{"", "unknown", other.RefreshToken.Reveal()} {
			for _, s := range must(e.Service.Sessions(t.Context(), user, token))(t) {
				if s.Current {
					t.Errorf("with %q, %v is current", token, s.ID)
				}
			}
		}
	})
}

func TestRevokeSession(t *testing.T) {
	e := apptest.New(t)
	user, session := signIn(t, e)
	stranger, _ := signIn(t, e)
	c, _ := newClient(t, e, user, "user:read")
	oauth := authorizationCodeTokens(t, e, user, c)

	for name, try := range map[string]struct{ user, id uuid.UUID }{
		"someone else's":  {stranger, session.Grant.ID},
		"an OAuth grant":  {user, oauth.Grant.ID},
		"one that is not": {user, uuid.NewV7()},
	} {
		t.Run(name, func(t *testing.T) {
			err := e.Service.RevokeSession(t.Context(), try.user, try.id)
			isKind(t, err, apperr.ErrNotFound)
			if apperr.Message(err) != "No such session." {
				t.Fatalf("message = %q", apperr.Message(err))
			}
		})
	}
	must(e.Store.Grant(t.Context(), session.Grant.ID, e.Clock.Now()))(t)

	if err := e.Service.RevokeSession(t.Context(), user, session.Grant.ID); err != nil {
		t.Fatal(err)
	}
	_, err := e.Service.RefreshSession(t.Context(), session.RefreshToken.Reveal())
	isKind(t, err, apperr.ErrUnauthorized)
}
