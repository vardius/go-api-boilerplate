package httpapi_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/vardius/go-api-boilerplate/internal/user/app"
	"github.com/vardius/go-api-boilerplate/internal/user/store"
)

// The callers every route is tried with, in this order: the owner goes
// last, as owning can use something up.
const (
	anonymous = iota
	client    // a client-credentials token: no user
	noScope   // the owner, without the scope the route needs
	other     // another user, with every user scope
	owner
	callers
)

var callerNames = [callers]string{"anonymous", "client", "noScope", "other", "owner"}

// access is one row of the matrix: a route and the status each caller
// gets from it.
type access struct {
	method, pattern string
	// path is the request's path and query; it defaults to the pattern.
	path string
	// scope is what noScope lacks.
	scope string
	// body returns a request body for the nth call.
	body func(m *matrix, n int) any
	want [callers]int
}

// matrix is the state a row's requests use.
type matrix struct {
	*rig
	owner, other store.User
	// ownerCode is an email-change code of the owner's.
	ownerCode string
}

// The authorization matrix: every route against every kind of caller. A
// route added without a row fails TestEveryRouteHasARow.
var rows = []access{
	{method: http.MethodGet, pattern: "/livez", want: all(http.StatusOK)},
	{method: http.MethodGet, pattern: "/readyz", want: all(http.StatusOK)},
	{method: http.MethodGet, pattern: "/v1/config", want: all(http.StatusOK)},
	{
		method: http.MethodPost, pattern: "/v1/login-links",
		body: func(_ *matrix, n int) any {
			return map[string]string{"email": fmt.Sprintf("someone-%d@example.com", n)}
		},
		want: all(http.StatusAccepted),
	},
	{
		method: http.MethodGet, pattern: "/v1/me", scope: app.ScopeUserRead,
		want: [callers]int{http.StatusUnauthorized, http.StatusForbidden, http.StatusForbidden, http.StatusOK, http.StatusOK},
	},
	{
		method: http.MethodPost, pattern: "/v1/me/email-change", scope: app.ScopeUserWrite,
		body: func(_ *matrix, n int) any { return map[string]string{"email": fmt.Sprintf("new-%d@example.com", n)} },
		want: [callers]int{http.StatusUnauthorized, http.StatusForbidden, http.StatusForbidden, http.StatusAccepted, http.StatusAccepted},
	},
	{
		method: http.MethodPost, pattern: "/v1/me/email-change/confirm", scope: app.ScopeUserWrite,
		body: func(m *matrix, _ int) any { return map[string]string{"code": m.ownerCode} },
		// Another user's code is no code at all to them.
		want: [callers]int{http.StatusUnauthorized, http.StatusForbidden, http.StatusForbidden, http.StatusBadRequest, http.StatusOK},
	},
	{
		method: http.MethodGet, pattern: "/v1/google/start", path: "/v1/google/start?link=1", scope: app.ScopeUserWrite,
		want: [callers]int{http.StatusUnauthorized, http.StatusForbidden, http.StatusForbidden, http.StatusFound, http.StatusFound},
	},
	{
		method: http.MethodGet, pattern: "/v1/google/start",
		want: all(http.StatusFound),
	},
	// Without its flow cookie, the callback only sends the browser back.
	{method: http.MethodGet, pattern: "/v1/google/callback", path: "/v1/google/callback?code=c&state=s", want: all(http.StatusFound)},
}

func all(status int) [callers]int {
	var s [callers]int
	for i := range s {
		s[i] = status
	}
	return s
}

func newMatrix(t *testing.T) *matrix {
	t.Helper()
	m := &matrix{rig: newRig(t, options{google: true})}
	m.owner = m.signUp("ada@example.com")
	m.other = m.signUp("eve@example.com")
	m.expect(m.as(m.ownerToken(m.owner)), http.MethodPost, "/v1/me/email-change", map[string]string{"email": "ada.lovelace@example.com"}, http.StatusAccepted)
	m.ownerCode = m.mailedCode("ada.lovelace@example.com")
	return m
}

func (m *matrix) caller(c int, scope string) *browser {
	all := []string{app.ScopeUserRead, app.ScopeUserWrite}
	switch c {
	case client:
		return m.as(m.clientToken())
	case noScope:
		var without []string
		for _, s := range all {
			if s != scope {
				without = append(without, s)
			}
		}
		return m.as(m.userToken(m.owner.ID, without...))
	case other:
		return m.as(m.userToken(m.other.ID, all...))
	case owner:
		return m.as(m.userToken(m.owner.ID, all...))
	default:
		return m.browser()
	}
}

func TestAuthorizationMatrix(t *testing.T) {
	m := newMatrix(t)
	n := 0
	for _, row := range rows {
		path := row.path
		if path == "" {
			path = row.pattern
		}
		for c := range callers {
			t.Run(fmt.Sprintf("%s %s as %s", row.method, path, callerNames[c]), func(t *testing.T) {
				n++
				var body any
				if row.body != nil {
					body = row.body(m, n)
				}
				res := m.do(m.caller(c, row.scope), row.method, path, body, nil)
				if res.status != row.want[c] {
					t.Fatalf("status = %d, want %d: %s", res.status, row.want[c], res.body)
				}
				if res.status == http.StatusOK && (row.pattern == "/v1/me" || row.pattern == "/v1/me/email-change/confirm") {
					var u struct {
						ID string `json:"id"`
					}
					res.decode(t, &u)
					want := m.owner.ID
					if c == other {
						want = m.other.ID
					}
					if u.ID != want.String() {
						t.Errorf("answered with user %s, want the caller %s", u.ID, want)
					}
				}
			})
		}
	}
}

func TestEveryRouteHasARow(t *testing.T) {
	m := newMatrix(t)
	covered := map[string]bool{}
	for _, row := range rows {
		covered[row.method+" "+row.pattern] = true
	}
	for _, route := range m.router.Routes() {
		if key := route.Method + " " + route.Pattern; !covered[key] {
			t.Errorf("%s has no row in the authorization matrix", key)
		}
	}
}
