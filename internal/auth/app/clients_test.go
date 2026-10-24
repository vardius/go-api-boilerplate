package app_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/auth/app"
	"github.com/vardius/go-api-boilerplate/internal/auth/app/apptest"
	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/config"
)

const redirectURI = "https://app.example.test/callback"

func newClient(t *testing.T, e *apptest.Env, owner uuid.UUID, scopes ...string) (store.Client, config.Secret) {
	t.Helper()
	c, secret, err := e.Service.CreateClient(t.Context(), owner, app.NewClient{
		Name:         "  The App ",
		RedirectURIs: []string{redirectURI, "http://localhost:9999/cb", redirectURI},
		Scopes:       scopes,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c, secret
}

func TestCreateClient(t *testing.T) {
	e := apptest.New(t)
	owner := uuid.NewV7()
	c, secret := newClient(t, e, owner, "clients:read", "user:read", "user:read")

	if c.Name != "The App" || strings.Join(c.RedirectURIs, " ") != redirectURI+" http://localhost:9999/cb" ||
		strings.Join(c.Scopes, " ") != "user:read clients:read" || c.OwnerID != owner || !c.CreatedAt.Equal(e.Clock.Now()) {
		t.Fatalf("created %+v", c)
	}
	if _, err := uuid.Parse(c.ID); err != nil {
		t.Fatalf("client ID %q: %v", c.ID, err)
	}
	if len(secret.Reveal()) != 43 {
		t.Fatalf("the secret is %d characters, want 43 (32 bytes in base64url)", len(secret.Reveal()))
	}

	stored := must(e.Store.Client(t.Context(), c.ID))(t)
	if !strings.HasPrefix(stored.SecretHash, "$argon2id$v=19$m=19456,t=2,p=1$") || strings.Contains(stored.SecretHash, secret.Reveal()) {
		t.Fatalf("stored secret = %q, want an argon2id hash", stored.SecretHash)
	}
	if got := must(e.Service.AuthenticateClient(t.Context(), c.ID, secret))(t); got.ID != c.ID {
		t.Fatalf("authenticated as %s", got.ID)
	}

	t.Run("two clients get different secrets", func(t *testing.T) {
		_, other := newClient(t, e, owner, "user:read")
		if other == secret {
			t.Fatal("the same secret twice")
		}
	})
}

func TestCreateClientValidates(t *testing.T) {
	e := apptest.New(t)
	ok := app.NewClient{Name: "App", RedirectURIs: []string{redirectURI}, Scopes: []string{"user:read"}}
	tests := []struct {
		name  string
		edit  func(*app.NewClient)
		field string
	}{
		{"no name", func(c *app.NewClient) { c.Name = "  " }, "name"},
		{"a long name", func(c *app.NewClient) { c.Name = strings.Repeat("é", 101) }, "name"},
		{"no redirect URI", func(c *app.NewClient) { c.RedirectURIs = nil }, "redirect_uris"},
		{"too many redirect URIs", func(c *app.NewClient) {
			c.RedirectURIs = nil
			for i := range 11 {
				c.RedirectURIs = append(c.RedirectURIs, fmt.Sprintf("https://app.example.test/%d", i))
			}
		}, "redirect_uris"},
		{"a relative URI", func(c *app.NewClient) { c.RedirectURIs = []string{"/callback"} }, "redirect_uris"},
		{"a URI that doesn't parse", func(c *app.NewClient) { c.RedirectURIs = []string{"https://app.example.test/%zz"} }, "redirect_uris"},
		{"an opaque URI", func(c *app.NewClient) { c.RedirectURIs = []string{"mailto:a@example.com"} }, "redirect_uris"},
		{"plain http", func(c *app.NewClient) { c.RedirectURIs = []string{"http://app.example.test/cb"} }, "redirect_uris"},
		{"http on a name ending in localhost", func(c *app.NewClient) { c.RedirectURIs = []string{"http://evil-localhost/cb"} }, "redirect_uris"},
		{"a custom scheme", func(c *app.NewClient) { c.RedirectURIs = []string{"com.example.app://cb"} }, "redirect_uris"},
		{"a fragment", func(c *app.NewClient) { c.RedirectURIs = []string{redirectURI + "#x"} }, "redirect_uris"},
		{"an empty fragment", func(c *app.NewClient) { c.RedirectURIs = []string{redirectURI + "#"} }, "redirect_uris"},
		{"credentials", func(c *app.NewClient) { c.RedirectURIs = []string{"https://user:pw@app.example.test/cb"} }, "redirect_uris"},
		{"a long URI", func(c *app.NewClient) { c.RedirectURIs = []string{redirectURI + "?" + strings.Repeat("a", 2000)} }, "redirect_uris"},
		{"no scopes", func(c *app.NewClient) { c.Scopes = nil }, "scopes"},
		{"the service scope", func(c *app.NewClient) { c.Scopes = []string{"login-codes:redeem"} }, "scopes"},
		{"an unknown scope", func(c *app.NewClient) { c.Scopes = []string{"user:read", "admin"} }, "scopes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := ok
			tt.edit(&in)
			_, _, err := e.Service.CreateClient(t.Context(), uuid.NewV7(), in)
			v, isValidation := errors.AsType[*apperr.ValidationError](err)
			if !isValidation || v.Fields[tt.field] == "" || len(v.Fields) != 1 {
				t.Fatalf("err = %v, want a problem with %s alone", err, tt.field)
			}
		})
	}

	t.Run("http on localhost and 127.0.0.1", func(t *testing.T) {
		in := ok
		in.RedirectURIs = []string{"http://localhost/cb", "http://127.0.0.1:8080/cb?x=1"}
		if _, _, err := e.Service.CreateClient(t.Context(), uuid.NewV7(), in); err != nil {
			t.Fatal(err)
		}
	})
}

func TestClientsListsTheOwnersOnly(t *testing.T) {
	e := apptest.New(t)
	owner := uuid.NewV7()
	for i := range 105 {
		c := store.Client{ID: uuid.NewV4().String(), OwnerID: owner, Name: fmt.Sprint(i), RedirectURIs: []string{redirectURI}, Scopes: []string{"user:read"}, SecretHash: "x", CreatedAt: e.Clock.Now()}
		if err := e.Store.CreateClient(t.Context(), c); err != nil {
			t.Fatal(err)
		}
		e.Clock.Advance(1)
	}
	newClient(t, e, uuid.NewV7(), "user:read")

	tests := []struct {
		limit, offset, want int
		first               string
	}{
		{20, 0, 20, "0"},
		{20, 100, 5, "100"},
		{1000, 0, app.MaxClientsPage, "0"},
		{0, -3, 1, "0"},
		{10, 200, 0, ""},
	}
	for _, tt := range tests {
		got := must(e.Service.Clients(t.Context(), owner, tt.limit, tt.offset))(t)
		if len(got) != tt.want || len(got) > 0 && got[0].Name != tt.first {
			t.Errorf("Clients(%d, %d) gave %d from %v, want %d from %s", tt.limit, tt.offset, len(got), got[0:min(1, len(got))], tt.want, tt.first)
		}
		for _, c := range got {
			if c.OwnerID != owner {
				t.Fatalf("listed another owner's client %s", c.ID)
			}
		}
	}
}

func TestDeleteClient(t *testing.T) {
	e := apptest.New(t)
	owner := uuid.NewV7()
	c, secret := newClient(t, e, owner, "user:read")
	tokens := authorizationCodeTokens(t, e, uuid.NewV7(), c)

	for name, try := range map[string]struct {
		owner uuid.UUID
		id    string
	}{
		"someone else's":     {uuid.NewV7(), c.ID},
		"a malformed ID":     {owner, "web"},
		"an unknown ID":      {owner, uuid.NewV4().String()},
		"a non-canonical ID": {owner, strings.ToUpper(c.ID)},
	} {
		t.Run(name, func(t *testing.T) {
			err := e.Service.DeleteClient(t.Context(), try.owner, try.id)
			isKind(t, err, apperr.ErrNotFound)
			if apperr.Message(err) != "No such client." {
				t.Fatalf("message = %q", apperr.Message(err))
			}
		})
	}

	if err := e.Service.DeleteClient(t.Context(), owner, c.ID); err != nil {
		t.Fatal(err)
	}
	_, err := e.Service.AuthenticateClient(t.Context(), c.ID, secret)
	oauthCode(t, err, app.ErrorInvalidClient)
	_, err = e.Store.Grant(t.Context(), tokens.Grant.ID, e.Clock.Now())
	isKind(t, err, apperr.ErrNotFound)
}

func TestAuthenticateClient(t *testing.T) {
	e := apptest.New(t)
	c, secret := newClient(t, e, uuid.NewV7(), "user:read")
	tests := map[string]struct {
		id     string
		secret config.Secret
	}{
		"no ID":             {"", secret},
		"no secret":         {c.ID, ""},
		"a malformed ID":    {"web", secret},
		"an unknown client": {uuid.NewV4().String(), secret},
		"a wrong secret":    {c.ID, secret + "x"},
		"a long secret":     {c.ID, config.Secret(strings.Repeat("a", 257))},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := e.Service.AuthenticateClient(t.Context(), tt.id, tt.secret)
			oauthCode(t, err, app.ErrorInvalidClient)
			noSecret(t, err, secret.Reveal(), tt.secret.Reveal())
		})
	}

	t.Run("a stored hash that isn't one", func(t *testing.T) {
		bad := store.Client{ID: uuid.NewV4().String(), OwnerID: uuid.NewV7(), Name: "x", RedirectURIs: []string{redirectURI}, Scopes: []string{"user:read"}, SecretHash: "plain"}
		if err := e.Store.CreateClient(t.Context(), bad); err != nil {
			t.Fatal(err)
		}
		_, err := e.Service.AuthenticateClient(t.Context(), bad.ID, "plain")
		if _, ok := errors.AsType[*app.OAuthError](err); err == nil || ok {
			t.Fatalf("err = %v, want a server error", err)
		}
	})
}
