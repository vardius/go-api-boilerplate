package httpapi_test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
	"uuid"
)

type clientJSON struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
}

const appRedirect = "http://localhost:9999/callback"

func createClient(t *testing.T, b *browser, scopes ...string) (clientJSON, string) {
	t.Helper()
	var created struct {
		Client       clientJSON `json:"client"`
		ClientSecret string     `json:"client_secret"`
	}
	b.expect(http.MethodPost, "/v1/clients", map[string]any{
		"name":          "Test app",
		"redirect_uris": []string{appRedirect},
		"scopes":        scopes,
	}, http.StatusCreated).decode(t, &created)
	if created.Client.ID == "" || created.ClientSecret == "" {
		t.Fatalf("created %+v", created)
	}
	return created.Client, created.ClientSecret
}

func TestClients(t *testing.T) {
	r := newRig(t, options{})
	owner := r.signIn(t, uuid.NewV7())
	c, secret := createClient(t, owner, "user:read", "clients:read")
	if c.Name != "Test app" || strings.Join(c.Scopes, " ") != "user:read clients:read" || c.RedirectURIs[0] != appRedirect {
		t.Fatalf("created %+v", c)
	}
	createClient(t, owner, "user:read")

	var list struct {
		Clients       []clientJSON `json:"clients"`
		Limit, Offset int
	}
	res := owner.expect(http.MethodGet, "/v1/clients?limit=1&offset=1", nil, http.StatusOK)
	res.decode(t, &list)
	if len(list.Clients) != 1 || list.Clients[0].ID == c.ID || list.Limit != 1 || list.Offset != 1 {
		t.Fatalf("second page = %s", res.body)
	}
	res = owner.expect(http.MethodGet, "/v1/clients", nil, http.StatusOK)
	res.decode(t, &list)
	if len(list.Clients) != 2 || list.Limit != 20 || list.Offset != 0 || bytes.Contains(res.body, []byte(secret)) || bytes.Contains(res.body, []byte("argon2")) {
		t.Fatalf("list = %s", res.body)
	}
	for _, q := range []string{"limit=0", "limit=101", "offset=-1", "limit=x"} {
		owner.expect(http.MethodGet, "/v1/clients?"+q, nil, http.StatusBadRequest)
	}

	t.Run("others see none of them and can't delete them", func(t *testing.T) {
		other := r.signIn(t, uuid.NewV7())
		var theirs struct{ Clients []clientJSON }
		other.expect(http.MethodGet, "/v1/clients", nil, http.StatusOK).decode(t, &theirs)
		if len(theirs.Clients) != 0 {
			t.Fatalf("another user lists %v", theirs.Clients)
		}
		other.expect(http.MethodDelete, "/v1/clients/"+c.ID, nil, http.StatusNotFound)
	})

	owner.expect(http.MethodDelete, "/v1/clients/"+c.ID, nil, http.StatusNoContent)
	owner.expect(http.MethodDelete, "/v1/clients/"+c.ID, nil, http.StatusNotFound)
}

func TestCreateClientValidates(t *testing.T) {
	r := newRig(t, options{})
	b := r.signIn(t, uuid.NewV7())
	res := b.expect(http.MethodPost, "/v1/clients", map[string]any{
		"name":          "",
		"redirect_uris": []string{"http://app.example.test/cb"},
		"scopes":        []string{"login-codes:redeem"},
	}, http.StatusBadRequest)
	var problem struct{ Errors map[string]string }
	res.decode(t, &problem)
	if len(problem.Errors) != 3 || problem.Errors["name"] == "" || problem.Errors["redirect_uris"] == "" || problem.Errors["scopes"] == "" {
		t.Fatalf("problem = %s", res.body)
	}
}
