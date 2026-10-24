package app_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/auth/app"
	"github.com/vardius/go-api-boilerplate/internal/auth/app/apptest"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
)

// must(f())(t) returns what f returns, failing t if f fails.
func must[T any](v T, err error) func(*testing.T) T {
	return func(t *testing.T) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func isKind(t *testing.T, err, kind error) {
	t.Helper()
	if !errors.Is(err, kind) {
		t.Fatalf("err = %v, want %v", err, kind)
	}
}

func oauthCode(t *testing.T, err error, code string) *app.OAuthError {
	t.Helper()
	oe, ok := errors.AsType[*app.OAuthError](err)
	if !ok || oe.Code != code {
		t.Fatalf("err = %v, want an OAuth %s", err, code)
	}
	return oe
}

// signIn starts a session for a new user and returns the user's ID with
// the session's tokens.
func signIn(t *testing.T, e *apptest.Env) (uuid.UUID, app.Tokens) {
	t.Helper()
	user := uuid.NewV7()
	code := "login-" + uuid.NewV4().String()
	e.Logins.Add(code, user, "someone@example.com")
	return user, must(e.Service.SignIn(t.Context(), code, "Mozilla/5.0"))(t)
}

func verify(t *testing.T, e *apptest.Env, token string) authn.Identity {
	t.Helper()
	return must(e.Verifier().Verify(context.WithoutCancel(t.Context()), token))(t)
}

// noSecret fails when err's text holds any of secrets.
func noSecret(t *testing.T, err error, secrets ...string) {
	t.Helper()
	if err == nil {
		return
	}
	if i := slices.IndexFunc(secrets, func(s string) bool { return s != "" && strings.Contains(err.Error(), s) }); i >= 0 {
		t.Fatalf("the error shows a secret: %v", err)
	}
}
