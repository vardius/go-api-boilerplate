package authn_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"uuid"

	"github.com/vardius/gorouter/v5"

	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn/authntest"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
)

// whoami answers with the caller's subject, or "anonymous".
var whoami = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	id, ok := authn.FromContext(r.Context())
	if !ok {
		_, _ = w.Write([]byte("anonymous"))
		return
	}
	_, _ = w.Write([]byte(id.Subject))
})

type call struct {
	header, cookie, query string
}

func (c call) request() *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/v1/me"+c.query, nil)
	if c.header != "" {
		r.Header.Set("Authorization", c.header)
	}
	if c.cookie != "" {
		r.AddCookie(&http.Cookie{Name: authn.AccessCookie, Value: c.cookie})
	}
	return r
}

func problemDetail(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var p struct {
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("not a problem: %q", rec.Body.String())
	}
	return p.Detail
}

func TestAuthenticate(t *testing.T) {
	iss := authntest.NewIssuer(t)
	user := uuid.NewV7()
	good := iss.Token(t, authn.Identity{UserID: user, Scopes: []string{"user:read"}})
	expired := iss.Token(t, authn.Identity{UserID: user}, authntest.WithExpiry(-10*time.Minute))

	tests := []struct {
		name   string
		call   call
		status int
		body   string
	}{
		{"bearer header", call{header: "Bearer " + good}, 200, user.String()},
		{"scheme is case-insensitive", call{header: "bearer " + good}, 200, user.String()},
		{"cookie", call{cookie: good}, 200, user.String()},
		{"no token", call{}, 200, "anonymous"},
		{"query string is ignored", call{query: "?access_token=" + good}, 200, "anonymous"},
		{"other scheme is left alone", call{header: "Basic d2ViOnNlY3JldA=="}, 200, "anonymous"},
		{"other scheme falls back to the cookie", call{header: "Basic d2ViOnNlY3JldA==", cookie: good}, 200, user.String()},
		{"bad token", call{header: "Bearer " + expired}, 401, ""},
		{"empty bearer", call{header: "Bearer "}, 401, ""},
		{"bad cookie", call{cookie: "garbage"}, 401, ""},
		{"header wins over cookie", call{header: "Bearer " + expired, cookie: good}, 401, ""},
	}
	h := authn.Authenticate(iss.Verifier())(whoami)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, tt.call.request())
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tt.status, rec.Body.String())
			}
			if tt.status == 401 {
				if rec.Header().Get("WWW-Authenticate") != "Bearer" {
					t.Errorf("WWW-Authenticate = %q", rec.Header().Get("WWW-Authenticate"))
				}
				return
			}
			if rec.Body.String() != tt.body {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.body)
			}
		})
	}
}

func TestRequire(t *testing.T) {
	iss := authntest.NewIssuer(t)
	session := iss.Token(t, authn.Identity{UserID: uuid.NewV7(), Scopes: []string{"user:read", "clients:read"}})
	oauthApp := iss.Token(t, authn.Identity{UserID: uuid.NewV7(), ClientID: "third-party", Scopes: []string{"user:read"}})
	service := iss.Token(t, authn.Identity{ClientID: "auth-service", Scopes: []string{"login-codes:redeem", "user:read"}})
	webItself := iss.Token(t, authn.Identity{ClientID: authn.FirstPartyClient, Scopes: []string{"user:read"}})

	tests := []struct {
		name   string
		guard  func(http.Handler) http.Handler
		token  string
		status int
		detail string
	}{
		{"anonymous", authn.Require(), "", 401, "This needs an access token."},
		{"any token", authn.Require(), service, 200, ""},
		{"has the scopes", authn.Require("user:read", "clients:read"), session, 200, ""},
		{"lacks a scope", authn.Require("user:read", "clients:write"), session, 403, "This needs the clients:write scope."},
		{"user route, user token", authn.RequireUser("user:read"), oauthApp, 200, ""},
		{"user route, client token", authn.RequireUser("user:read"), service, 403, "This needs a user's access token."},
		{"user route, missing scope", authn.RequireUser("user:write"), session, 403, "This needs the user:write scope."},
		{"user route, anonymous", authn.RequireUser(), "", 401, "This needs an access token."},
		{"first party, session", authn.RequireFirstParty(), session, 200, ""},
		{"first party, OAuth app", authn.RequireFirstParty(), oauthApp, 403, "This needs a signed-in session."},
		{"first party, web client's own token", authn.RequireFirstParty(), webItself, 403, "This needs a signed-in session."},
		{"first party, anonymous", authn.RequireFirstParty(), "", 401, "This needs an access token."},
	}
	v := iss.Verifier()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := authn.Authenticate(v)(tt.guard(whoami))
			rec := httptest.NewRecorder()
			c := call{}
			if tt.token != "" {
				c.header = "Bearer " + tt.token
			}
			h.ServeHTTP(rec, c.request())
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tt.status, rec.Body.String())
			}
			if tt.detail != "" {
				if got := problemDetail(t, rec); got != tt.detail {
					t.Errorf("detail = %q, want %q", got, tt.detail)
				}
			}
		})
	}
}

// TestAuthenticateKeepsTheRoutedPattern checks that Log, outside
// Authenticate, still sees the pattern gorouter sets.
func TestAuthenticateKeepsTheRoutedPattern(t *testing.T) {
	iss := authntest.NewIssuer(t)
	router := gorouter.New()
	router.GET("/v1/clients/{id}", whoami)

	var logs bytes.Buffer
	h := httpx.Chain(httpx.Log(slog.New(slog.NewJSONHandler(&logs, nil))), authn.Authenticate(iss.Verifier()))(router)
	r := httptest.NewRequest(http.MethodGet, "/v1/clients/7", nil)
	r.Header.Set("Authorization", "Bearer "+iss.Token(t, authn.Identity{UserID: uuid.NewV7()}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)

	var line struct{ Pattern string }
	if err := json.Unmarshal(logs.Bytes(), &line); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 || line.Pattern != "/v1/clients/{id}" {
		t.Errorf("status %d, logged pattern %q", rec.Code, line.Pattern)
	}
}

func TestFromContextAnonymous(t *testing.T) {
	if id, ok := authn.FromContext(t.Context()); ok {
		t.Errorf("FromContext = %+v", id)
	}
}
