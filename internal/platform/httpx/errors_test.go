package httpx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/es"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
	"github.com/vardius/go-api-boilerplate/internal/platform/obs"
)

func serve(h http.Handler, r *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func readProblem(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("Content-Type = %q, body %q", ct, rec.Body.String())
	}
	var p map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("problem isn't JSON: %v: %q", err, rec.Body.String())
	}
	return p
}

func failWith(err error) http.Handler {
	return httpx.Handle(func(http.ResponseWriter, *http.Request) error { return err })
}

func TestWriteErrorTable(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		detail string
		fields map[string]any
	}{
		{"validation", fmt.Errorf("register: %w", apperr.Invalid("email", "is not an address")), 400, "", map[string]any{"email": "is not an address"}},
		{"invalid with message", fmt.Errorf("parse: %w", apperr.New(apperr.ErrInvalid, "The code is malformed.")), 400, "The code is malformed.", nil},
		{"bare invalid", apperr.ErrInvalid, 400, "", nil},
		{"body too large", fmt.Errorf("read: %w", &http.MaxBytesError{Limit: 1024}), 413, "The request body is larger than 1024 bytes.", nil},
		{"too many requests", fmt.Errorf("login link: %w", &httpx.TooManyRequestsError{RetryAfter: 59500 * time.Millisecond}), 429, "Too many requests; try again in 1m0s.", nil},
		{"unauthorized", fmt.Errorf("verify token: %w", apperr.ErrUnauthorized), 401, "", nil},
		{"forbidden", apperr.New(apperr.ErrForbidden, "This needs the user:write scope."), 403, "This needs the user:write scope.", nil},
		{"not found", fmt.Errorf("load user 42: %w", apperr.ErrNotFound), 404, "", nil},
		{"conflict", fmt.Errorf("append: %w", es.ErrConcurrency), 409, "this was changed by another request; reload and try again", nil},
		{"deadline", fmt.Errorf("query: %w", context.DeadlineExceeded), 503, "", nil},
		{"internal", errors.New("dial mysql: password=hunter2: connection refused"), 500, "", nil},
		{"internal with a message", fmt.Errorf("%w: %w", errors.New("boom"), apperr.New(errors.New("custom kind"), "leaked text")), 500, "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/v1/users?x=1", nil)
			r = r.WithContext(obs.WithRequestID(r.Context(), "req-7"))
			rec := serve(failWith(tt.err), r)

			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
			p := readProblem(t, rec)
			want := map[string]any{
				"type":       "about:blank",
				"title":      http.StatusText(tt.status),
				"status":     float64(tt.status),
				"instance":   "/v1/users",
				"request_id": "req-7",
			}
			if tt.detail != "" {
				want["detail"] = tt.detail
			}
			if tt.fields != nil {
				want["errors"] = tt.fields
			}
			if !equalJSON(p, want) {
				t.Errorf("problem = %v\nwant      %v", p, want)
			}
			if got := rec.Header().Get("WWW-Authenticate"); (tt.status == 401) != (got == "Bearer") {
				t.Errorf("WWW-Authenticate = %q", got)
			}
			if strings.Contains(rec.Body.String(), "hunter2") || strings.Contains(rec.Body.String(), "leaked") {
				t.Errorf("internal error text reached the client: %s", rec.Body.String())
			}
		})
	}
}

func equalJSON(a, b map[string]any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y)
}

func TestWriteErrorCanceledWritesNothing(t *testing.T) {
	rec := serve(failWith(fmt.Errorf("query: %w", context.Canceled)), httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Body.Len() != 0 || rec.Header().Get("Content-Type") != "" {
		t.Errorf("wrote %q with %v", rec.Body.String(), rec.Header())
	}
}

// useDefaultLogger points slog's default logger at a buffer for one test.
func useDefaultLogger(t *testing.T) *bytes.Buffer {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func TestWriteErrorLogsWithoutLogMiddleware(t *testing.T) {
	logs := useDefaultLogger(t)
	serve(failWith(errors.New("disk full")), httptest.NewRequest(http.MethodGet, "/", nil))
	serve(failWith(context.Canceled), httptest.NewRequest(http.MethodGet, "/", nil))
	serve(failWith(apperr.ErrNotFound), httptest.NewRequest(http.MethodGet, "/", nil))

	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("logged %d lines, want 2:\n%s", len(lines), logs.String())
	}
	if !strings.Contains(lines[0], `"level":"ERROR"`) || !strings.Contains(lines[0], `"err":"disk full"`) {
		t.Errorf("500 line = %s", lines[0])
	}
	if !strings.Contains(lines[1], `"level":"DEBUG"`) || !strings.Contains(lines[1], "context canceled") {
		t.Errorf("canceled line = %s", lines[1])
	}
}

func TestWriteErrorAfterTheResponseStarted(t *testing.T) {
	var logs bytes.Buffer
	h := httpx.Log(slog.New(slog.NewJSONHandler(&logs, nil)))(httpx.Handle(func(w http.ResponseWriter, _ *http.Request) error {
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("id,email\n1,"))
		return errors.New("scan row 2: connection reset")
	}))
	rec := serve(h, httptest.NewRequest(http.MethodGet, "/export", nil))

	if rec.Code != 200 || rec.Body.String() != "id,email\n1," {
		t.Errorf("response = %d %q, want the partial body untouched", rec.Code, rec.Body.String())
	}
	if !strings.Contains(logs.String(), "scan row 2: connection reset") {
		t.Errorf("error not logged: %s", logs.String())
	}
}

func TestWriteJSON(t *testing.T) {
	t.Run("keeps the status with a nil value", func(t *testing.T) {
		rec := serve(httpx.Handle(func(w http.ResponseWriter, _ *http.Request) error {
			return httpx.WriteJSON(w, http.StatusCreated, nil)
		}), httptest.NewRequest(http.MethodPost, "/", nil))
		if rec.Code != http.StatusCreated || rec.Body.String() != "null\n" || rec.Header().Get("Content-Type") != "application/json" {
			t.Errorf("got %d %q %v", rec.Code, rec.Body.String(), rec.Header())
		}
	})
	t.Run("encodes nil slices as arrays", func(t *testing.T) {
		rec := serve(httpx.Handle(func(w http.ResponseWriter, _ *http.Request) error {
			return httpx.WriteJSON(w, http.StatusOK, struct {
				Clients []string `json:"clients"`
			}{})
		}), httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Body.String() != `{"clients":[]}`+"\n" {
			t.Errorf("body = %q", rec.Body.String())
		}
	})
	t.Run("an encoding failure becomes a 500 before anything is written", func(t *testing.T) {
		rec := serve(httpx.Handle(func(w http.ResponseWriter, _ *http.Request) error {
			return httpx.WriteJSON(w, http.StatusOK, map[string]any{"f": func() {}})
		}), httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != 500 {
			t.Errorf("status = %d", rec.Code)
		}
		readProblem(t, rec)
	})
}
