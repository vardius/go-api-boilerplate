package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"uuid"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	legacyrouter "github.com/getkin/kin-openapi/routers/legacy"
	"github.com/vardius/gorouter/v5"

	"github.com/vardius/go-api-boilerplate/internal/auth/app"
	"github.com/vardius/go-api-boilerplate/internal/auth/app/apptest"
	"github.com/vardius/go-api-boilerplate/internal/auth/httpapi"
	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
	"github.com/vardius/go-api-boilerplate/internal/platform/run"
)

const (
	openAPIPath = "../../../api/auth.openapi.yaml"
	webURL      = "https://web.example.test"
	// The tests' server is the API itself, not the gateway, so the
	// refresh cookie's path is the API's own.
	refreshPath = "/v1/sessions"
)

type options struct {
	insecureCookies bool
	rateLimit       httpx.RateLimitConfig
	// store replaces the Env's for the Service, to make it fail.
	store func(store.Store) store.Store
}

type rig struct {
	e      *apptest.Env
	router *gorouter.Router
	srv    *httptest.Server
	logs   *logBuffer
	api    routers.Router
}

type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func newRig(t *testing.T, o options) *rig {
	t.Helper()
	e := apptest.New(t)
	if o.rateLimit == (httpx.RateLimitConfig{}) {
		o.rateLimit = httpx.RateLimitConfig{Rate: 1000, Burst: 1000}
	}
	service := e.Service
	if o.store != nil {
		service = app.New(o.store(e.Store), e.Signer, e.Logins.Client, e.Clock.Now)
	}
	logs := &logBuffer{}
	router, err := httpapi.New(httpapi.Config{
		App:               service,
		Keys:              e.Keys,
		Probe:             run.NewProbe(nil),
		Verifier:          e.Verifier(),
		Issuer:            apptest.Issuer,
		WebURL:            webURL,
		CookieSecure:      !o.insecureCookies,
		RefreshCookiePath: refreshPath,
		RateLimit:         o.rateLimit,
		Logger:            slog.New(slog.NewJSONHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewTLSServer(router)
	t.Cleanup(srv.Close)

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(openAPIPath)
	if err != nil {
		t.Fatal(err)
	}
	doc.Servers = openapi3.Servers{{URL: srv.URL}}
	api, err := legacyrouter.NewRouter(doc)
	if err != nil {
		t.Fatal(err)
	}
	return &rig{e: e, router: router, srv: srv, logs: logs, api: api}
}

// validating checks every response against the OpenAPI document.
type validating struct {
	t    *testing.T
	base http.RoundTripper
	api  routers.Router
}

func (v validating) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := v.base.RoundTrip(req)
	if err != nil {
		return res, err
	}
	body, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		return nil, err
	}
	res.Body = io.NopCloser(bytes.NewReader(body))

	route, params, err := v.api.FindRoute(req)
	if err != nil {
		v.t.Errorf("%s %s is not in the OpenAPI document: %v", req.Method, req.URL.Path, err)
		return res, nil
	}
	in := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req, PathParams: params, Route: route},
		Status:                 res.StatusCode,
		Header:                 res.Header,
		Options:                &openapi3filter.Options{IncludeResponseStatus: true},
	}
	in.SetBodyBytes(body)
	if err := openapi3filter.ValidateResponse(req.Context(), in); err != nil {
		v.t.Errorf("%s %s answered %d against the OpenAPI document: %v\n%s", req.Method, req.URL.Path, res.StatusCode, err, body)
	}
	return res, nil
}

// browser is a client with its own cookie jar that doesn't follow
// redirects, and checks every response against the OpenAPI document.
type browser struct {
	t      *testing.T
	r      *rig
	client *http.Client
	// header goes on every request.
	header http.Header
}

func (r *rig) browser(t *testing.T) *browser {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &browser{t: t, r: r, client: r.httpClient(t, jar)}
}

// httpClient returns a client of the rig's server that validates its
// responses.
func (r *rig) httpClient(t *testing.T, jar http.CookieJar) *http.Client {
	return &http.Client{
		Jar:           jar,
		Transport:     validating{t: t, base: r.srv.Client().Transport, api: r.api},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

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

func (res response) location(t *testing.T) *url.URL {
	t.Helper()
	u, err := url.Parse(res.header.Get("Location"))
	if err != nil || res.header.Get("Location") == "" {
		t.Fatalf("no Location in a %d: %v", res.status, err)
	}
	return u
}

// oauthError returns the error code of an RFC 6749 error response.
func (res response) oauthError(t *testing.T) string {
	t.Helper()
	var e struct {
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	res.decode(t, &e)
	return e.Error
}

func (b *browser) send(req *http.Request) response {
	b.t.Helper()
	res, err := b.client.Do(req)
	if err != nil {
		b.t.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		b.t.Fatal(err)
	}
	return response{status: res.StatusCode, header: res.Header, body: body}
}

func (b *browser) request(method, path string, body io.Reader, header http.Header) *http.Request {
	b.t.Helper()
	req, err := http.NewRequestWithContext(b.t.Context(), method, b.r.srv.URL+path, body)
	if err != nil {
		b.t.Fatal(err)
	}
	for k, v := range b.header {
		req.Header[k] = v
	}
	for k, v := range header {
		req.Header[k] = v
	}
	return req
}

// do sends body, if any, as JSON.
func (b *browser) do(method, path string, body any, header http.Header) response {
	b.t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			b.t.Fatal(err)
		}
		reader = bytes.NewReader(data)
	}
	req := b.request(method, path, reader, header)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return b.send(req)
}

func (b *browser) expect(method, path string, body any, want int) response {
	b.t.Helper()
	res := b.do(method, path, body, nil)
	if res.status != want {
		b.t.Fatalf("%s %s = %d, want %d: %s", method, path, res.status, want, res.body)
	}
	return res
}

// form posts form to path.
func (b *browser) form(path string, form url.Values, header http.Header) response {
	b.t.Helper()
	req := b.request(http.MethodPost, path, strings.NewReader(form.Encode()), header)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return b.send(req)
}

// cookie returns the jar's cookie name for path.
func (b *browser) cookie(name, path string) string {
	u, _ := url.Parse(b.r.srv.URL + path)
	for _, c := range b.client.Jar.Cookies(u) {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

// setCookie puts a cookie in the jar.
func (b *browser) setCookie(name, value, path string) {
	u, _ := url.Parse(b.r.srv.URL + path)
	b.client.Jar.SetCookies(u, []*http.Cookie{{Name: name, Value: value, Path: path, Secure: true}})
}

// signIn signs a new browser in as user.
func (r *rig) signIn(t *testing.T, user uuid.UUID) *browser {
	t.Helper()
	b := r.browser(t)
	code := "code-" + uuid.NewV4().String()
	r.e.Logins.Add(code, user, "someone@example.com")
	b.expect(http.MethodPost, "/v1/sessions", map[string]string{"code": code}, http.StatusCreated)
	return b
}

// setCookies returns the response's cookies by name.
func setCookies(res response) map[string]*http.Cookie {
	out := map[string]*http.Cookie{}
	for _, c := range (&http.Response{Header: res.header}).Cookies() {
		out[c.Name] = c
	}
	return out
}
