package httpapi_test

import (
	"net/http"
	"net/url"
	"testing"
	"uuid"

	"golang.org/x/oauth2"

	"github.com/vardius/go-api-boilerplate/internal/auth/app"
	"github.com/vardius/go-api-boilerplate/internal/auth/app/apptest"
	"github.com/vardius/go-api-boilerplate/internal/auth/keys"
	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
)

// The callers of the matrix, in its columns' order.
var callers = []string{
	"anonymous",
	"another user",
	"the owner",
	"a token without the scope",
	"a client's own token",
	"a third-party app's user token",
}

// fixture is what one cell of the matrix runs against: the owner, with a
// second session, an OAuth client and a pending authorization request.
type fixture struct {
	r       *rig
	owner   uuid.UUID
	session uuid.UUID
	client  store.Client
	secret  string
	request uuid.UUID
	callers map[string]*browser
}

// registered is a client secret and its hash, made once: hashing is slow
// on purpose.
type registered struct{ secret, hash string }

func newFixture(t *testing.T, reg registered) *fixture {
	t.Helper()
	r := newRig(t, options{})
	e := r.e
	f := &fixture{r: r, owner: uuid.NewV7(), secret: reg.secret}
	f.client = store.Client{
		ID:           uuid.NewV4().String(),
		OwnerID:      f.owner,
		Name:         "App",
		RedirectURIs: []string{"https://app.example.test/cb"},
		Scopes:       []string{app.ScopeUserRead, app.ScopeClientsRead, app.ScopeClientsWrite},
		SecretHash:   reg.hash,
		CreatedAt:    e.Clock.Now(),
	}
	if err := e.Store.CreateClient(t.Context(), f.client); err != nil {
		t.Fatal(err)
	}

	second := r.signIn(t, f.owner)
	var list struct {
		Sessions []struct{ ID uuid.UUID } `json:"sessions"`
	}
	second.expect(http.MethodGet, "/v1/sessions", nil, http.StatusOK).decode(t, &list)
	f.session = list.Sessions[0].ID

	a, err := e.Service.CheckAuthorization(t.Context(), f.authorizeQuery())
	if err != nil {
		t.Fatal(err)
	}
	if f.request, err = e.Service.RequestConsent(t.Context(), f.owner, a); err != nil {
		t.Fatal(err)
	}

	bearer := func(subject, client string, scopes ...string) *browser {
		token, err := e.Signer.Sign(keys.Claims{Subject: subject, ClientID: client, Scopes: scopes, IssuedAt: e.Clock.Now()})
		if err != nil {
			t.Fatal(err)
		}
		b := r.browser(t)
		b.header = http.Header{"Authorization": {"Bearer " + token}}
		return b
	}
	f.callers = map[string]*browser{
		"anonymous":                      r.browser(t),
		"another user":                   r.signIn(t, uuid.NewV7()),
		"the owner":                      r.signIn(t, f.owner),
		"a token without the scope":      bearer(f.owner.String(), authn.FirstPartyClient),
		"a client's own token":           bearer(f.client.ID, f.client.ID, f.client.Scopes...),
		"a third-party app's user token": bearer(f.owner.String(), f.client.ID, f.client.Scopes...),
	}
	return f
}

func (f *fixture) authorizeQuery() url.Values {
	return url.Values{
		"response_type":         {"code"},
		"client_id":             {f.client.ID},
		"redirect_uri":          {f.client.RedirectURIs[0]},
		"state":                 {"s"},
		"code_challenge":        {oauth2.S256ChallengeFromVerifier(oauth2.GenerateVerifier())},
		"code_challenge_method": {"S256"},
	}
}

// basic is the client's HTTP Basic credentials, which replace the
// caller's own Authorization header.
func (f *fixture) basic() http.Header {
	req := &http.Request{Header: http.Header{}}
	req.SetBasicAuth(url.QueryEscape(f.client.ID), url.QueryEscape(f.secret))
	return req.Header
}

type cell = [6]int

func all(status int) cell { return cell{status, status, status, status, status, status} }

// The authorization matrix: what every route answers each kind of caller.
// A route without a row fails the test.
func TestAuthorizationMatrix(t *testing.T) {
	setup := apptest.New(t)
	c, secret, err := setup.Service.CreateClient(t.Context(), uuid.NewV7(), app.NewClient{Name: "x", RedirectURIs: []string{"https://app.example.test/cb"}, Scopes: []string{app.ScopeUserRead}})
	if err != nil {
		t.Fatal(err)
	}
	stored, err := setup.Store.Client(t.Context(), c.ID)
	if err != nil {
		t.Fatal(err)
	}
	reg := registered{secret: secret.Reveal(), hash: stored.SecretHash}

	rows := []struct {
		method, pattern string
		send            func(f *fixture, b *browser) response
		want            cell
	}{
		{"GET", "/livez", get("/livez"), all(200)},
		{"GET", "/readyz", get("/readyz"), all(200)},
		{"GET", "/.well-known/jwks.json", get("/.well-known/jwks.json"), all(200)},
		{"GET", "/.well-known/oauth-authorization-server", get("/.well-known/oauth-authorization-server"), all(200)},
		{"POST", "/v1/sessions", func(f *fixture, b *browser) response {
			code := "code-" + uuid.NewV4().String()
			f.r.e.Logins.Add(code, uuid.NewV7(), "a@example.com")
			return b.do(http.MethodPost, "/v1/sessions", map[string]string{"code": code}, nil)
		}, all(201)},
		{"GET", "/v1/sessions", get("/v1/sessions"), cell{401, 200, 200, 200, 403, 403}},
		{"POST", "/v1/sessions/refresh", func(_ *fixture, b *browser) response {
			return b.do(http.MethodPost, "/v1/sessions/refresh", nil, nil)
		}, cell{401, 200, 200, 401, 401, 401}},
		{"DELETE", "/v1/sessions/current", func(_ *fixture, b *browser) response {
			return b.do(http.MethodDelete, "/v1/sessions/current", nil, nil)
		}, all(204)},
		{"DELETE", "/v1/sessions/{id}", func(f *fixture, b *browser) response {
			return b.do(http.MethodDelete, "/v1/sessions/"+f.session.String(), nil, nil)
		}, cell{401, 404, 204, 204, 403, 403}},
		{"GET", "/v1/clients", get("/v1/clients"), cell{401, 200, 200, 403, 403, 200}},
		{"POST", "/v1/clients", func(_ *fixture, b *browser) response {
			return b.do(http.MethodPost, "/v1/clients", map[string]any{"name": "New", "redirect_uris": []string{"https://new.example.test/cb"}, "scopes": []string{"user:read"}}, nil)
		}, cell{401, 201, 201, 403, 403, 201}},
		{"DELETE", "/v1/clients/{id}", func(f *fixture, b *browser) response {
			return b.do(http.MethodDelete, "/v1/clients/"+f.client.ID, nil, nil)
		}, cell{401, 404, 204, 403, 403, 204}},
		{"GET", "/v1/authorization-requests/{id}", func(f *fixture, b *browser) response {
			return b.do(http.MethodGet, "/v1/authorization-requests/"+f.request.String(), nil, nil)
		}, cell{401, 404, 200, 200, 403, 403}},
		// The OAuth pages answer by redirecting; where to is tested on its own.
		{"GET", "/oauth/authorize", func(f *fixture, b *browser) response {
			return b.do(http.MethodGet, "/oauth/authorize?"+f.authorizeQuery().Encode(), nil, nil)
		}, all(302)},
		{"POST", "/oauth/authorize", func(f *fixture, b *browser) response {
			return b.form("/oauth/authorize", url.Values{"request": {f.request.String()}, "decision": {"allow"}}, nil)
		}, all(303)},
		// The client authenticates itself, so whoever else the caller is
		// makes no difference.
		{"POST", "/oauth/token", func(f *fixture, b *browser) response {
			return b.form("/oauth/token", url.Values{"grant_type": {"client_credentials"}}, f.basic())
		}, all(200)},
		{"POST", "/oauth/revoke", func(f *fixture, b *browser) response {
			return b.form("/oauth/revoke", url.Values{"token": {"unknown"}}, f.basic())
		}, all(200)},
	}

	served := map[string]bool{}
	for _, route := range newRig(t, options{}).router.Routes() {
		served[route.Method+" "+route.Pattern] = true
	}
	for _, row := range rows {
		route := row.method + " " + row.pattern
		if !served[route] {
			t.Errorf("the matrix has %s, which is not served", route)
		}
		delete(served, route)
		for i, caller := range callers {
			t.Run(route+" by "+caller, func(t *testing.T) {
				f := newFixture(t, reg)
				b := f.callers[caller]
				b.t = t
				if res := row.send(f, b); res.status != row.want[i] {
					t.Errorf("%s by %s = %d, want %d: %s", route, caller, res.status, row.want[i], res.body)
				}
			})
		}
	}
	for route := range served {
		t.Errorf("%s has no row in the authorization matrix", route)
	}
}

func get(path string) func(*fixture, *browser) response {
	return func(_ *fixture, b *browser) response { return b.do(http.MethodGet, path, nil, nil) }
}
