package obs_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/vardius/go-api-boilerplate/internal/platform/config"
	"github.com/vardius/go-api-boilerplate/internal/platform/obs"
)

func decodeLines(t *testing.T, b *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for line := range strings.Lines(b.String()) {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("not JSON: %q: %v", line, err)
		}
		lines = append(lines, m)
	}
	return lines
}

func TestLoggerAddsContextIDs(t *testing.T) {
	var out bytes.Buffer
	log := obs.NewLogger(&out, obs.Config{Level: slog.LevelInfo})

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x4b, 0xf9, 0x2f, 0x35, 0x77, 0xb3, 0x4d, 0xa6, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36},
		SpanID:     trace.SpanID{0x00, 0xf0, 0x67, 0xaa, 0x0b, 0xa9, 0x02, 0xb7},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := obs.WithRequestID(trace.ContextWithSpanContext(t.Context(), sc), "req-1")
	log.InfoContext(ctx, "served")
	log.With("component", "relay").WithGroup("job").InfoContext(t.Context(), "idle", "n", 1)
	log.DebugContext(ctx, "dropped below the level")

	lines := decodeLines(t, &out)
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %s", len(lines), out.String())
	}
	if got := lines[0]["trace_id"]; got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("trace_id = %v", got)
	}
	if got := lines[0]["span_id"]; got != "00f067aa0ba902b7" {
		t.Errorf("span_id = %v", got)
	}
	if got := lines[0]["request_id"]; got != "req-1" {
		t.Errorf("request_id = %v", got)
	}
	if _, ok := lines[1]["trace_id"]; ok {
		t.Errorf("a context without a span added a trace_id: %v", lines[1])
	}
	if lines[1]["component"] != "relay" || lines[1]["job"].(map[string]any)["n"] != 1.0 {
		t.Errorf("attrs and groups lost: %v", lines[1])
	}
}

func TestLoggerRedactsCredentials(t *testing.T) {
	var out bytes.Buffer
	log := obs.NewLogger(&out, obs.Config{})

	log.With("token", "t0").Info("login",
		"password", "p1",
		"Authorization", "Bearer t2",
		"set-cookie", "access_token=t3",
		"client_secret", "s4",
		"code", "c5",
		slog.Group("headers", "Cookie", "refresh_token=t6", "Accept", "application/json"),
		"email", "ada@example.com",
	)

	text := out.String()
	for _, secret := range []string{"t0", "p1", "t2", "t3", "s4", "c5", "t6"} {
		if strings.Contains(text, `"`+secret) || strings.Contains(text, " "+secret) || strings.Contains(text, "="+secret) {
			t.Errorf("%s leaked: %s", secret, text)
		}
	}
	line := decodeLines(t, &out)[0]
	if line["email"] != "ada@example.com" || line["headers"].(map[string]any)["Accept"] != "application/json" {
		t.Errorf("redacted too much: %v", line)
	}
	if line["Authorization"] != "[redacted]" {
		t.Errorf("Authorization = %v", line["Authorization"])
	}
}

func TestTextFormat(t *testing.T) {
	var out bytes.Buffer
	obs.NewLogger(&out, obs.Config{Format: obs.FormatText}).Info("hello", "password", "p1")
	if got := out.String(); !strings.Contains(got, "msg=hello") || !strings.Contains(got, "password=[redacted]") {
		t.Errorf("text output = %q", got)
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("LEVEL", "debug")
	t.Setenv("FORMAT", "text")
	cfg, err := config.Load[obs.Config]()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Level != slog.LevelDebug || cfg.Format != obs.FormatText {
		t.Errorf("cfg = %+v", cfg)
	}

	t.Setenv("FORMAT", "yaml")
	if _, err := config.Load[obs.Config](); err == nil || !strings.Contains(err.Error(), `log format "yaml"`) {
		t.Errorf("a bad format loaded: %v", err)
	}
}

func TestRequestIDAbsent(t *testing.T) {
	if got := obs.RequestID(t.Context()); got != "" {
		t.Errorf("RequestID = %q", got)
	}
}

// collector is an OTLP/HTTP endpoint that records the paths posted to it.
type collector struct {
	mu    sync.Mutex
	paths []string
}

func (c *collector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	c.paths = append(c.paths, r.Method+" "+r.URL.Path)
	c.mu.Unlock()
	w.Header().Set("Content-Type", "application/x-protobuf")
}

func (c *collector) posted() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Compact(slices.Sorted(slices.Values(c.paths)))
}

func emit(ctx context.Context, t *testing.T) {
	t.Helper()
	_, span := otel.Tracer("test").Start(ctx, "work")
	span.End()
	counter, err := otel.Meter("test").Int64Counter("work.done")
	if err != nil {
		t.Fatal(err)
	}
	counter.Add(ctx, 1)
}

func resetGlobals(t *testing.T) {
	t.Cleanup(func() {
		if _, err := obs.Setup(context.Background(), "test", "dev"); err != nil {
			t.Error(err)
		}
	})
}

func TestSetupExportsWhenEndpointSet(t *testing.T) {
	resetGlobals(t)
	c := &collector{}
	srv := httptest.NewServer(c)
	defer srv.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", srv.URL)

	shutdown, err := obs.Setup(t.Context(), "user", "v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	emit(t.Context(), t)
	if err := shutdown(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got, want := c.posted(), []string{"POST /v1/metrics", "POST /v1/traces"}; !slices.Equal(got, want) {
		t.Errorf("collector got %v, want %v", got, want)
	}
	if fields := otel.GetTextMapPropagator().Fields(); !slices.Contains(fields, "traceparent") || !slices.Contains(fields, "baggage") {
		t.Errorf("propagator fields = %v", fields)
	}
}

func TestSetupPerSignalEndpoint(t *testing.T) {
	resetGlobals(t)
	c := &collector{}
	srv := httptest.NewServer(c)
	defer srv.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", srv.URL+"/custom/traces")

	shutdown, err := obs.Setup(t.Context(), "auth", "v1")
	if err != nil {
		t.Fatal(err)
	}
	emit(t.Context(), t)
	if err := shutdown(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got, want := c.posted(), []string{"POST /custom/traces"}; !slices.Equal(got, want) {
		t.Errorf("collector got %v, want %v", got, want)
	}
}

func TestSetupIsNoopWithoutEndpoint(t *testing.T) {
	resetGlobals(t)
	shutdown, err := obs.Setup(t.Context(), "user", "v1")
	if err != nil {
		t.Fatal(err)
	}
	_, span := otel.Tracer("test").Start(t.Context(), "work")
	defer span.End()
	if span.IsRecording() {
		t.Error("span records with no endpoint configured")
	}
	if err := shutdown(t.Context()); err != nil {
		t.Error(err)
	}
}

func TestSetupRejectsBadResourceAttributes(t *testing.T) {
	resetGlobals(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:1")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "not-a-pair")
	if _, err := obs.Setup(t.Context(), "user", "v1"); err == nil {
		t.Error("Setup accepted malformed OTEL_RESOURCE_ATTRIBUTES")
	}
}
