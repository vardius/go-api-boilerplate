package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/vardius/gorouter/v5"

	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn/authntest"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
	"github.com/vardius/go-api-boilerplate/internal/platform/memdb"
	"github.com/vardius/go-api-boilerplate/internal/platform/run"
	"github.com/vardius/go-api-boilerplate/internal/user/app"
	"github.com/vardius/go-api-boilerplate/internal/user/httpapi"
	"github.com/vardius/go-api-boilerplate/internal/user/mail"
	"github.com/vardius/go-api-boilerplate/internal/user/store"
	"github.com/vardius/go-api-boilerplate/internal/user/store/memory"
)

const openAPIPath = "../../../api/user.openapi.yaml"

var t0 = time.Date(2026, 10, 24, 9, 30, 15, 0, time.UTC)

// webURL is where the web app lives in these tests.
const webURL = "https://app.example.com"

type rig struct {
	t      *testing.T
	iss    *authntest.Issuer
	app    *app.Service
	outbox *memdb.Outbox
	router *gorouter.Router
	srv    *httptest.Server
	spec   routers.Router
	google *fakeGoogle // nil when Google sign-in is off

	mu  sync.Mutex
	now time.Time
}

type options struct {
	google    bool
	preview   string
	rateLimit httpx.RateLimitConfig
	// redirectURL is the Google callback's public URL.
	redirectURL string
}

func newRig(t *testing.T, opts options) *rig {
	t.Helper()
	db := memdb.New()
	r := &rig{t: t, iss: authntest.NewIssuer(t), outbox: memdb.NewOutbox(db), now: t0}
	web, _ := url.Parse(webURL)
	r.app = app.New(app.Deps{
		UnitOfWork: db,
		Events:     memdb.NewEventStore(db),
		Outbox:     r.outbox,
		Users:      memory.NewUsers(db),
		LoginCodes: memory.NewLoginCodes(db),
		WebURL:     web,
		Now:        r.clock,
	})
	cfg := httpapi.Config{
		App:            r.app,
		Probe:          run.NewProbe(nil),
		Verifier:       r.iss.Verifier(),
		MailPreviewURL: opts.preview,
		RateLimit:      opts.rateLimit,
		Logger:         slog.New(slog.DiscardHandler),
		Now:            r.clock,
	}
	if opts.google {
		r.google = newFakeGoogle(t, r.clock)
		redirect := opts.redirectURL
		if redirect == "" {
			redirect = "http://localhost:8080/api/user/v1/google/callback"
		}
		g, err := httpapi.NewGoogle(httpapi.GoogleConfig{ClientID: googleClientID, ClientSecret: googleClientSecret, RedirectURL: redirect}, r.google.issuer(), r.google.client())
		if err != nil {
			t.Fatal(err)
		}
		g.Now = r.clock
		cfg.Google = g
	}
	r.router = httpapi.New(cfg)
	r.srv = httptest.NewServer(r.router)
	t.Cleanup(r.srv.Close)
	r.spec = loadSpec(t, r.srv.URL)
	return r
}

func (r *rig) clock() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.now
}

func (r *rig) advance(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.now = r.now.Add(d)
}

// loadSpec reads the OpenAPI document, served from base.
func loadSpec(t *testing.T, base string) routers.Router {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromFile(openAPIPath)
	if err != nil {
		t.Fatal(err)
	}
	doc.Servers = openapi3.Servers{{URL: base}}
	router, err := legacy.NewRouter(doc)
	if err != nil {
		t.Fatal(err)
	}
	return router
}

// response is what a test gets back, already checked against the OpenAPI
// document.
type response struct {
	status int
	header http.Header
	body   []byte
}

func (res response) decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(res.body, v); err != nil {
		t.Fatalf("decode %s: %v", res.body, err)
	}
}

// problem is the RFC 9457 body of an error.
type problem struct {
	Status int               `json:"status"`
	Detail string            `json:"detail"`
	Errors map[string]string `json:"errors"`
}

func (res response) problem(t *testing.T) problem {
	t.Helper()
	if ct := res.header.Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want a problem: %s", ct, res.body)
	}
	var p problem
	res.decode(t, &p)
	return p
}

func (res response) location(t *testing.T) *url.URL {
	t.Helper()
	u, err := url.Parse(res.header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// browser is a client with a cookie jar that doesn't follow redirects.
type browser struct {
	client *http.Client
	token  string
}

func (r *rig) browser() *browser {
	jar, _ := cookiejar.New(nil)
	return &browser{client: &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// as returns a browser sending token as its bearer.
func (r *rig) as(token string) *browser {
	b := r.browser()
	b.token = token
	return b
}

type header = map[string]string

// do sends a request, with body encoded as JSON unless it is a string, and
// checks the response against the OpenAPI document.
func (r *rig) do(b *browser, method, path string, body any, h header) response {
	r.t.Helper()
	var reader io.Reader
	switch body := body.(type) {
	case nil:
	case string:
		reader = strings.NewReader(body)
	default:
		data, err := json.Marshal(body)
		if err != nil {
			r.t.Fatal(err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(r.t.Context(), method, r.srv.URL+path, reader)
	if err != nil {
		r.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if b.token != "" {
		req.Header.Set("Authorization", "Bearer "+b.token)
	}
	for k, v := range h {
		req.Header.Set(k, v)
	}
	res, err := b.client.Do(req)
	if err != nil {
		r.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		r.t.Fatal(err)
	}
	r.conforms(req, res, data)
	return response{status: res.StatusCode, header: res.Header, body: data}
}

// conforms fails the test when res isn't a response the OpenAPI document
// gives req.
func (r *rig) conforms(req *http.Request, res *http.Response, body []byte) {
	r.t.Helper()
	route, params, err := r.spec.FindRoute(req)
	if err != nil {
		r.t.Errorf("%s %s is not in %s: %v", req.Method, req.URL.Path, openAPIPath, err)
		return
	}
	in := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req, PathParams: params, Route: route},
		Status:                 res.StatusCode,
		Header:                 res.Header,
		Options:                &openapi3filter.Options{IncludeResponseStatus: true, MultiError: true},
	}
	in.SetBodyBytes(body)
	if err := openapi3filter.ValidateResponse(context.WithoutCancel(r.t.Context()), in); err != nil {
		r.t.Errorf("%s %s → %d doesn't match %s: %v\n%s", req.Method, req.URL.Path, res.StatusCode, openAPIPath, err, body)
	}
}

func (r *rig) expect(b *browser, method, path string, body any, want int) response {
	r.t.Helper()
	res := r.do(b, method, path, body, nil)
	if res.status != want {
		r.t.Fatalf("%s %s = %d, want %d: %s", method, path, res.status, want, res.body)
	}
	return res
}

// The identities tests call with.

func (r *rig) userToken(id uuid.UUID, scopes ...string) string {
	return r.iss.Token(r.t, authn.Identity{Subject: id.String(), UserID: id, Scopes: scopes})
}

func (r *rig) ownerToken(u store.User) string {
	return r.userToken(u.ID, app.ScopeUserRead, app.ScopeUserWrite)
}

func (r *rig) clientToken() string {
	return r.iss.Token(r.t, authn.Identity{Subject: "app-1", ClientID: "app-1", Scopes: []string{app.ScopeUserRead, app.ScopeUserWrite}})
}

var authService = authn.NewContext(context.Background(), authn.Identity{Subject: "auth", ClientID: "auth", Scopes: []string{app.ScopeRedeemLoginCodes}})

// mailed takes the messages waiting in the outbox.
func (r *rig) mailed() []mail.Link {
	r.t.Helper()
	msgs, err := r.outbox.Claim(r.t.Context(), r.clock(), time.Minute, 100)
	if err != nil {
		r.t.Fatal(err)
	}
	links := make([]mail.Link, len(msgs))
	for i, m := range msgs {
		if err := json.Unmarshal(m.Payload, &links[i]); err != nil {
			r.t.Fatal(err)
		}
		if err := r.outbox.Ack(r.t.Context(), m.ID); err != nil {
			r.t.Fatal(err)
		}
	}
	return links
}

// mailedCode takes the one message waiting and returns the code in its
// link.
func (r *rig) mailedCode(to string) string {
	r.t.Helper()
	links := r.mailed()
	if len(links) != 1 || links[0].To != to {
		r.t.Fatalf("mailed %+v, want one link to %s", links, to)
	}
	u, err := url.Parse(links[0].URL)
	if err != nil {
		r.t.Fatal(err)
	}
	return u.Query().Get("code")
}

// signUp registers email the way a browser and the auth service would.
func (r *rig) signUp(email string) store.User {
	r.t.Helper()
	r.expect(r.browser(), http.MethodPost, "/v1/login-links", map[string]string{"email": email}, http.StatusAccepted)
	u, err := r.app.RedeemLoginCode(authService, r.mailedCode(email))
	if err != nil {
		r.t.Fatal(err)
	}
	return u
}
