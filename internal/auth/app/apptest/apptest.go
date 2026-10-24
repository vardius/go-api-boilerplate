// Package apptest runs the auth use cases for tests: a memory store, real
// signing keys, a clock the test moves, and a fake user.v1.LoginService
// served over HTTP that checks auth's service token as the user service
// does.
package apptest

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
	"uuid"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	userv1 "github.com/vardius/go-api-boilerplate/gen/user/v1"
	"github.com/vardius/go-api-boilerplate/gen/user/v1/userv1connect"
	"github.com/vardius/go-api-boilerplate/internal/auth/app"
	"github.com/vardius/go-api-boilerplate/internal/auth/keys"
	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/auth/store/memory"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/connectx"
	"github.com/vardius/go-api-boilerplate/internal/platform/memdb"
)

// The issuer and audience of the tokens an Env signs.
const (
	Issuer   = "https://auth.example.test/api/auth"
	Audience = "go-api-boilerplate"
)

// Env is a Service with everything it runs on.
type Env struct {
	Service *app.Service
	Store   store.Store
	Keys    *keys.Keys
	Signer  *keys.Signer
	Logins  *Logins
	Clock   *Clock
}

// New returns an Env whose clock starts at the real time, truncated to the
// millisecond.
func New(t testing.TB) *Env {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	k, err := keys.New(key)
	if err != nil {
		t.Fatal(err)
	}
	e := &Env{
		Store:  memory.New(memdb.New()),
		Keys:   k,
		Signer: keys.NewSigner(k, Issuer, Audience),
		Clock:  &Clock{now: time.Now().UTC().Truncate(time.Millisecond)},
	}
	e.Logins = newLogins(t, e.Signer, e.Clock.Now)
	e.Service = app.New(e.Store, e.Signer, e.Logins.Client, e.Clock.Now)
	return e
}

// Verifier returns a verifier of the Env's tokens that reads the Env's
// clock.
func (e *Env) Verifier() *authn.Verifier {
	v := e.Signer.Verifier()
	v.Now = e.Clock.Now
	return v
}

// Clock is a time that moves only when told to.
type Clock struct {
	mu  sync.Mutex
	now time.Time
}

// Now returns the clock's time.
func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the clock on by d.
func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// Logins is a fake user.v1.LoginService. It redeems each code it was given
// once, and refuses a call without auth's service token, as the user
// service does.
type Logins struct {
	// Client calls the fake with auth's service token.
	Client userv1connect.LoginServiceClient
	// URL is where the fake is served, for a client of the test's own.
	URL string

	mu    sync.Mutex
	codes map[string]login
	fail  error
	calls int
}

func newLogins(t testing.TB, signer *keys.Signer, now func() time.Time) *Logins {
	l := &Logins{codes: map[string]login{}}
	v := signer.Verifier()
	v.Now = now
	srv := connect.NewServer(
		connectx.Errors(slog.New(slog.DiscardHandler)),
		connectx.Authenticate(v, keys.ServiceScope),
	)
	userv1connect.RegisterLoginServiceHandler(srv, l)
	mux := http.NewServeMux()
	connecthttp.Mount(mux, srv)
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	l.URL = hs.URL
	tokens := keys.NewServiceToken(signer, now)
	l.Client = userv1connect.NewLoginServiceClient(connect.NewClient(connecthttp.NewTransport(hs.Client(), hs.URL), connectx.Bearer(tokens.Token)))
	return l
}

type login struct{ user, email string }

// AddRaw makes code redeemable once, answering with user as the user ID,
// whatever it holds.
func (l *Logins) AddRaw(code, user string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.codes[code] = login{user: user}
}

// Add makes code redeemable once, signing in user with email.
func (l *Logins) Add(code string, user uuid.UUID, email string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.codes[code] = login{user: user.String(), email: email}
}

// FailWith makes every call fail with err, an error the user service could
// return; nil makes calls work again.
func (l *Logins) FailWith(err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fail = err
}

// Calls returns how many calls passed the service token check.
func (l *Logins) Calls() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.calls
}

// RedeemLoginCode implements userv1connect.LoginServiceHandler.
func (l *Logins) RedeemLoginCode(_ context.Context, req *userv1.RedeemLoginCodeRequest) (*userv1.RedeemLoginCodeResponse, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls++
	if l.fail != nil {
		return nil, l.fail
	}
	res, ok := l.codes[req.GetCode()]
	if !ok {
		return nil, apperr.New(apperr.ErrUnauthorized, "The code is invalid, used or expired.")
	}
	delete(l.codes, req.GetCode())
	return &userv1.RedeemLoginCodeResponse{UserId: res.user, Email: res.email}, nil
}
