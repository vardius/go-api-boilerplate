package httpx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/vardius/gorouter/v5"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
	"github.com/vardius/go-api-boilerplate/internal/platform/obs"
)

type logBuffer struct{ bytes.Buffer }

func newLog() (*slog.Logger, *logBuffer) {
	var b logBuffer
	return obs.NewLogger(&b, obs.Config{Level: slog.LevelDebug}), &b
}

func (b *logBuffer) lines(t *testing.T) []map[string]any {
	t.Helper()
	var out []map[string]any
	for line := range strings.Lines(b.String()) {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line isn't JSON: %q", line)
		}
		out = append(out, m)
	}
	return out
}

func TestChain(t *testing.T) {
	var calls []string
	mw := func(name string) httpx.Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, name+">")
				next.ServeHTTP(w, r)
				calls = append(calls, "<"+name)
			})
		}
	}
	h := httpx.Chain(mw("a"), mw("b"))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls = append(calls, "h") }))
	serve(h, httptest.NewRequest(http.MethodGet, "/", nil))
	if want := []string{"a>", "b>", "h", "<b", "<a"}; !slices.Equal(calls, want) {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}

func TestRecover(t *testing.T) {
	t.Run("before the response starts", func(t *testing.T) {
		log, logs := newLog()
		h := httpx.Chain(httpx.Recover(log), httpx.RequestID)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic("nil map write")
		}))
		r := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
		r.Header.Set("X-Request-Id", "req-9")
		rec := serve(h, r)

		if rec.Code != 500 {
			t.Fatalf("status = %d", rec.Code)
		}
		if p := readProblem(t, rec); p["request_id"] != "req-9" || p["detail"] != nil {
			t.Errorf("problem = %v", p)
		}
		line := logs.lines(t)[0]
		if line["msg"] != "handler panicked" || line["panic"] != "nil map write" || line["request_id"] != "req-9" || !strings.Contains(line["stack"].(string), "TestRecover") {
			t.Errorf("log = %v", line)
		}
	})
	t.Run("after the response started", func(t *testing.T) {
		log, _ := newLog()
		h := httpx.Recover(log)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"items":[`))
			panic("boom")
		}))
		rec := httptest.NewRecorder()
		v := catch(func() { h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil)) })
		if v != http.ErrAbortHandler {
			t.Errorf("panic = %v, want http.ErrAbortHandler so the connection is cut", v)
		}
		if got := rec.Body.String(); got != `{"items":[` {
			t.Errorf("body = %q, want nothing appended", got)
		}
	})
	t.Run("ErrAbortHandler passes through", func(t *testing.T) {
		log, logs := newLog()
		h := httpx.Recover(log)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic(fmt.Errorf("client gone: %w", http.ErrAbortHandler))
		}))
		v := catch(func() { serve(h, httptest.NewRequest(http.MethodGet, "/", nil)) })
		if err, ok := v.(error); !ok || !errors.Is(err, http.ErrAbortHandler) {
			t.Errorf("panic = %v", v)
		}
		if logs.Len() != 0 {
			t.Errorf("logged an intended abort: %s", logs.String())
		}
	})
}

func catch(f func()) (v any) {
	defer func() { v = recover() }()
	f()
	return nil
}

func TestRequestID(t *testing.T) {
	tests := []struct {
		incoming string
		keep     bool
	}{
		{"req-1.A_b", true},
		{strings.Repeat("a", 64), true},
		{"", false},
		{strings.Repeat("a", 65), false},
		{"has space", false},
		{"inject\nnewline", false},
		{`"quoted"`, false},
	}
	for _, tt := range tests {
		t.Run(tt.incoming, func(t *testing.T) {
			var seen string
			h := httpx.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen = obs.RequestID(r.Context()) }))
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header["X-Request-Id"] = []string{tt.incoming}
			rec := serve(h, r)

			echoed := rec.Header().Get("X-Request-Id")
			if echoed != seen {
				t.Errorf("echoed %q, handler saw %q", echoed, seen)
			}
			if tt.keep {
				if seen != tt.incoming {
					t.Errorf("replaced a valid ID with %q", seen)
				}
				return
			}
			if id, err := uuid.Parse(seen); err != nil || id[6]>>4 != 7 {
				t.Errorf("new ID %q isn't a UUIDv7", seen)
			}
		})
	}
}

func TestLog(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		level   string
		status  float64
		bytes   float64
		err     string
	}{
		{"ok", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("hello")) }, "INFO", 200, 5, ""},
		{"nothing written", func(http.ResponseWriter, *http.Request) {}, "INFO", 200, 0, ""},
		{"interim then final", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusEarlyHints)
			w.WriteHeader(http.StatusCreated)
		}, "INFO", 201, 0, ""},
		{"client error", httpx.Handle(func(http.ResponseWriter, *http.Request) error {
			return fmt.Errorf("load user 7: %w", apperr.ErrNotFound)
		}), "INFO", 404, -1, "load user 7: not found"},
		{"server error", httpx.Handle(func(http.ResponseWriter, *http.Request) error {
			return errors.New("dial tcp: connection refused")
		}), "ERROR", 500, -1, "dial tcp: connection refused"},
		{"client left", httpx.Handle(func(http.ResponseWriter, *http.Request) error {
			return fmt.Errorf("query: %w", context.Canceled)
		}), "DEBUG", 200, 0, "query: context canceled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, logs := newLog()
			r := httptest.NewRequest(http.MethodDelete, "/v1/sessions/1", nil)
			r = r.WithContext(obs.WithRequestID(r.Context(), "req-3"))
			serve(httpx.Log(log)(tt.handler), r)

			lines := logs.lines(t)
			if len(lines) != 1 {
				t.Fatalf("logged %d lines", len(lines))
			}
			l := lines[0]
			if l["msg"] != "request" || l["level"] != tt.level || l["method"] != "DELETE" || l["status"] != tt.status || l["request_id"] != "req-3" {
				t.Errorf("line = %v", l)
			}
			if _, ok := l["duration"].(float64); !ok {
				t.Errorf("no duration: %v", l)
			}
			if tt.bytes >= 0 && l["bytes"] != tt.bytes {
				t.Errorf("bytes = %v, want %v", l["bytes"], tt.bytes)
			}
			if got, _ := l["err"].(string); got != tt.err {
				t.Errorf("err = %q, want %q", got, tt.err)
			}
		})
	}
}

func TestLogRecordsPanics(t *testing.T) {
	log, logs := newLog()
	h := httpx.Chain(httpx.Recover(log), httpx.Log(log))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	rec := serve(h, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != 500 {
		t.Errorf("status = %d", rec.Code)
	}
	lines := logs.lines(t)
	if len(lines) != 2 {
		t.Fatalf("lines = %v", lines)
	}
	if l := lines[0]; l["msg"] != "request" || l["status"] != 500.0 || l["panic"] != true || l["level"] != "ERROR" {
		t.Errorf("request line = %v", l)
	}
}

func TestSecurityHeaders(t *testing.T) {
	rec := serve(httpx.SecurityHeaders(http.NotFoundHandler()), httptest.NewRequest(http.MethodGet, "/", nil))
	want := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "no-referrer",
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
		"Cache-Control":           "no-store",
	}
	for k, v := range want {
		if got := rec.Header().Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
}

func TestBodyLimit(t *testing.T) {
	var original io.ReadCloser
	restored := false
	outer := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			original = r.Body
			next.ServeHTTP(w, r)
			restored = r.Body == original
		})
	}
	h := httpx.Chain(outer, httpx.BodyLimit(8))(httpx.Handle(func(_ http.ResponseWriter, r *http.Request) error {
		_, err := io.ReadAll(r.Body)
		return err
	}))
	rec := serve(h, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("more than eight bytes")))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d", rec.Code)
	}
	if !restored {
		t.Error("the original body wasn't put back")
	}
	if rec := serve(h, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("short"))); rec.Code != 200 {
		t.Errorf("a small body got %d", rec.Code)
	}
}

func TestCrossOrigin(t *testing.T) {
	h := httpx.CrossOrigin(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	tests := []struct {
		name    string
		method  string
		headers map[string]string
		want    int
	}{
		{"same-origin post", http.MethodPost, map[string]string{"Sec-Fetch-Site": "same-origin"}, 204},
		{"cross-site post", http.MethodPost, map[string]string{"Sec-Fetch-Site": "cross-site"}, 403},
		{"cross-site get", http.MethodGet, map[string]string{"Sec-Fetch-Site": "cross-site"}, 204},
		{"foreign origin", http.MethodDelete, map[string]string{"Origin": "https://evil.example"}, 403},
		{"no browser headers", http.MethodPost, nil, 204},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "https://app.example/api/auth/v1/sessions", nil)
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}
			rec := serve(h, r)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
			if tt.want == 403 {
				if p := readProblem(t, rec); p["detail"] != "Cross-origin requests are not allowed." {
					t.Errorf("problem = %v", p)
				}
			}
		})
	}
}

func TestRecorderReachesTheConnection(t *testing.T) {
	log, _ := newLog()
	h := httpx.Chain(httpx.Recover(log), httpx.Trace, httpx.Log(log))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		rc := http.NewResponseController(w)
		if err := rc.SetWriteDeadline(time.Now().Add(time.Minute)); err != nil {
			http.Error(w, "SetWriteDeadline: "+err.Error(), 500)
			return
		}
		f, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "not a Flusher", 500)
			return
		}
		_, _ = io.WriteString(w, "chunk")
		f.Flush()
		if err := rc.Flush(); err != nil {
			http.Error(w, "Flush: "+err.Error(), 500)
		}
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()

	res, err := srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || string(body) != "chunk" {
		t.Errorf("got %d %q", res.StatusCode, body)
	}
}

func TestFlushBeforeWriteCountsAsStarted(t *testing.T) {
	log, logs := newLog()
	h := httpx.Log(log)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.(http.Flusher).Flush()
		httpx.WriteError(w, httptest.NewRequest(http.MethodGet, "/", nil), errors.New("late"))
	}))
	rec := serve(h, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Body.Len() != 0 || rec.Code != 200 {
		t.Errorf("wrote %d %q after the response started", rec.Code, rec.Body.String())
	}
	if l := logs.lines(t)[0]; l["err"] != "late" {
		t.Errorf("line = %v", l)
	}
}

// TestRoutedPatternReachesTraceAndLog runs the whole chain around gorouter,
// with a middleware that adds to the context in between, as authn does.
func TestRoutedPatternReachesTraceAndLog(t *testing.T) {
	spans := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans)))
	t.Cleanup(func() { otel.SetTracerProvider(tracenoop.NewTracerProvider()) })

	type key struct{}
	addsToContext := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			httpx.ServeWithContext(context.WithValue(r.Context(), key{}, "v"), next, w, r)
		})
	}
	router := gorouter.New()
	router.GET("/v1/clients/{id}", httpx.Handle(func(_ http.ResponseWriter, r *http.Request) error {
		if r.Context().Value(key{}) != "v" {
			return errors.New("context value lost")
		}
		return fmt.Errorf("load client %s: %w", r.PathValue("id"), apperr.ErrNotFound)
	}))

	log, logs := newLog()
	h := httpx.Chain(
		httpx.Recover(log),
		httpx.RequestID,
		httpx.Trace,
		httpx.Log(log),
		httpx.SecurityHeaders,
		addsToContext,
		httpx.CrossOrigin,
	)(router)
	rec := serve(h, httptest.NewRequest(http.MethodGet, "/v1/clients/42", nil))

	if rec.Code != 404 {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	line := logs.lines(t)[0]
	if line["pattern"] != "/v1/clients/{id}" || line["err"] != "load client 42: not found" {
		t.Errorf("log line = %v", line)
	}
	if line["trace_id"] == nil || line["request_id"] != rec.Header().Get("X-Request-Id") {
		t.Errorf("log line lacks the trace or request ID: %v", line)
	}
	ended := spans.Ended()
	if len(ended) != 1 || ended[0].Name() != "GET /v1/clients/{id}" {
		names := []string{}
		for _, s := range ended {
			names = append(names, s.Name())
		}
		t.Errorf("spans = %v, want one named GET /v1/clients/{id}", names)
	}
}

func TestNewServer(t *testing.T) {
	srv := httpx.NewServer(httpx.ServerConfig{Addr: ":9000", ReadTimeout: time.Second}, http.NotFoundHandler())
	if srv.Addr != ":9000" || srv.ReadTimeout != time.Second || srv.Handler == nil {
		t.Errorf("server = %+v", srv)
	}
	if srv.ReadHeaderTimeout != 5*time.Second || srv.WriteTimeout != 15*time.Second || srv.IdleTimeout != time.Minute || srv.MaxHeaderBytes != 64<<10 {
		t.Errorf("zero fields didn't take the defaults: %+v", srv)
	}
	if srv := httpx.NewServer(httpx.ServerConfig{}, nil); srv.Addr != ":8080" {
		t.Errorf("Addr = %q", srv.Addr)
	}
}
