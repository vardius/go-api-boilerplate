package run

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
)

// Check reports whether a dependency the process needs, such as its
// database, is usable.
type Check func(ctx context.Context) error

// Probe answers the liveness and readiness probes of an orchestrator.
type Probe struct {
	checks   map[string]Check
	draining atomic.Bool
}

// NewProbe returns a Probe whose readiness depends on checks, by name.
func NewProbe(checks map[string]Check) *Probe {
	return &Probe{checks: maps.Clone(checks)}
}

func (p *Probe) drain() {
	if p != nil {
		p.draining.Store(true)
	}
}

// Live answers 200 for as long as the process can serve at all.
func (p *Probe) Live() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = io.WriteString(w, "ok\n")
	})
}

// Ready answers 200 when every check passes, and a 503 problem naming the
// checks that fail, or saying the process is draining, otherwise. Check
// errors are logged, not sent, since probes are often reachable from
// outside.
func (p *Probe) Ready() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p.draining.Load() {
			unavailable(w, r, "The service is shutting down.")
			return
		}
		var failed []string
		for _, name := range slices.Sorted(maps.Keys(p.checks)) {
			if err := p.checks[name](r.Context()); err != nil {
				slog.WarnContext(r.Context(), "readiness check failed", "check", name, "err", err)
				failed = append(failed, name)
			}
		}
		if len(failed) > 0 {
			unavailable(w, r, "Failing checks: "+strings.Join(failed, ", ")+".")
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = io.WriteString(w, "ready\n")
	})
}

// unavailable writes the RFC 9457 problem httpx writes for its errors.
func unavailable(w http.ResponseWriter, r *http.Request, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusServiceUnavailable)
	_ = json.NewEncoder(w).Encode(struct {
		Type     string `json:"type"`
		Title    string `json:"title"`
		Status   int    `json:"status"`
		Detail   string `json:"detail"`
		Instance string `json:"instance"`
	}{"about:blank", http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable, detail, r.URL.Path})
}
