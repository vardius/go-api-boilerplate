// Package obs sets up logs, traces and metrics.
//
// NewLogger builds the process logger. Every record logged with a context
// carries the trace and span IDs of the OTel span in it and the request ID
// set by WithRequestID, so a log line leads to its trace. Values under keys
// that name credentials are replaced with "[redacted]" wherever they appear.
//
// Setup installs the OTel tracer and meter providers, exporting over
// OTLP/HTTP when the standard OTEL_EXPORTER_OTLP_* variables name an
// endpoint, and no-op providers otherwise.
package obs

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/vardius/golog/v2"
	"go.opentelemetry.io/otel/trace"
)

// Config sets up the logger. Services nest it with envPrefix:"LOG_".
type Config struct {
	Level  slog.Level `env:"LEVEL" envDefault:"info" desc:"Lowest level logged: debug, info, warn or error."`
	Format Format     `env:"FORMAT" envDefault:"json" desc:"Log format: json or text."`
}

// Format is how log records are written.
type Format string

// The formats. The zero Format is FormatJSON.
const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

// UnmarshalText accepts "json" or "text", so a typo fails at startup rather
// than silently picking a format.
func (f *Format) UnmarshalText(text []byte) error {
	switch v := Format(text); v {
	case FormatJSON, FormatText:
		*f = v
		return nil
	}
	return fmt.Errorf("log format %q: want json or text", text)
}

// NewLogger returns a logger writing cfg.Format records of at least
// cfg.Level to w.
func NewLogger(w io.Writer, cfg Config) *slog.Logger {
	var h slog.Handler = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: cfg.Level, ReplaceAttr: redact})
	if cfg.Format == FormatText {
		h = golog.NewHandler(w, &golog.Options{Level: cfg.Level, ReplaceAttr: redact})
	}
	return slog.New(contextHandler{h})
}

// credentialKeys are compared after lower-casing and turning "-" into "_".
var credentialKeys = map[string]struct{}{
	"password":      {},
	"secret":        {},
	"token":         {},
	"access_token":  {},
	"refresh_token": {},
	"authorization": {},
	"cookie":        {},
	"set_cookie":    {},
	"client_secret": {},
	"code":          {},
	"code_verifier": {},
}

func redact(_ []string, a slog.Attr) slog.Attr {
	key := strings.ReplaceAll(strings.ToLower(a.Key), "-", "_")
	if _, ok := credentialKeys[key]; ok {
		return slog.String(a.Key, "[redacted]")
	}
	return a
}

// contextHandler adds the IDs carried by a record's context. They land in
// the logger's current group, if it has one.
type contextHandler struct {
	slog.Handler
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(slog.String("trace_id", sc.TraceID().String()), slog.String("span_id", sc.SpanID().String()))
	}
	if id := RequestID(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}

type requestIDKey struct{}

// WithRequestID returns ctx carrying the request ID id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestID returns the request ID in ctx, or "" when there is none.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}
