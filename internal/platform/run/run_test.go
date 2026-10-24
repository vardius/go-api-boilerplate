package run_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"testing/synctest"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/platform/run"
)

var discard = slog.New(slog.DiscardHandler)

// TestMain starts os/signal's runtime machinery outside any synctest bubble.
// The runtime creates its signal channels on the first Notify; when that
// happens inside a bubble they belong to it, and the next Notify from
// outside, or from another bubble, is a fatal error.
func TestMain(m *testing.M) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR1)
	signal.Stop(c)
	os.Exit(m.Run())
}

// timeline records what happened when, in the bubble's fake time.
type timeline struct {
	start  time.Time
	mu     sync.Mutex
	events []string
}

func newTimeline() *timeline { return &timeline{start: time.Now()} }

func (l *timeline) add(event string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, time.Since(l.start).String()+" "+event)
}

func (l *timeline) get() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.events)
}

// recorded logs a server's Shutdown and the end of its Serve.
type recorded struct {
	name string
	run.Server
	log *timeline
}

func (s recorded) Serve() error {
	err := s.Server.Serve()
	s.log.add(s.name + " served")
	return err
}

func (s recorded) Shutdown(ctx context.Context) error {
	s.log.add(s.name + " shutdown")
	return s.Server.Shutdown(ctx)
}

func untilDone(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func status(p *run.Probe) int {
	rec := httptest.NewRecorder()
	p.Ready().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	return rec.Code
}

func TestRunDrainsBeforeShuttingDown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log := newTimeline()
		probe := run.NewProbe(nil)
		ctx, cancel := context.WithCancel(t.Context())
		cfg := run.Config{ShutdownDelay: 5 * time.Second, ShutdownTimeout: 20 * time.Second}

		var err error
		done := make(chan struct{})
		go func() {
			defer close(done)
			err = run.Run(ctx, cfg, probe, discard, recorded{"worker", run.Worker(untilDone), log})
		}()

		time.Sleep(time.Minute)
		synctest.Wait()
		if got := status(probe); got != http.StatusOK {
			t.Fatalf("ready while serving = %d", got)
		}

		cancel()
		synctest.Wait()
		if got := status(probe); got != http.StatusServiceUnavailable {
			t.Errorf("ready while draining = %d, want 503", got)
		}
		if got := log.get(); len(got) != 0 {
			t.Errorf("servers stopped before the delay: %v", got)
		}

		<-done
		if err != nil {
			t.Errorf("Run = %v", err)
		}
		if got, want := log.get(), []string{"1m5s worker shutdown", "1m5s worker served"}; !slices.Equal(got, want) {
			t.Errorf("events = %v, want %v", got, want)
		}
	})
}

func TestRunStopsEveryServerWhenOneFails(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log := newTimeline()
		probe := run.NewProbe(nil)
		cfg := run.Config{ShutdownDelay: time.Second, ShutdownTimeout: time.Second}
		boom := errors.New("bind: address already in use")

		err := run.Run(t.Context(), cfg, probe, discard,
			recorded{"failing", run.Worker(func(context.Context) error { return boom }), log},
			recorded{"healthy", run.Worker(untilDone), log},
		)

		if !errors.Is(err, boom) {
			t.Errorf("Run = %v, want %v", err, boom)
		}
		if strings.Contains(err.Error(), "healthy") {
			t.Errorf("a clean stop was reported as a failure: %v", err)
		}
		events := log.get()
		if !slices.Contains(events, "1s healthy shutdown") || !slices.Contains(events, "1s healthy served") {
			t.Errorf("the healthy server wasn't stopped after the delay: %v", events)
		}
		if got := status(probe); got != http.StatusServiceUnavailable {
			t.Errorf("ready after a failure = %d, want 503", got)
		}
	})
}

func TestRunReportsServersThatOutliveTheTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		stuck := run.Worker(func(context.Context) error {
			<-release
			return nil
		})
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := run.Run(ctx, run.Config{ShutdownTimeout: 20 * time.Second}, nil, discard, stuck)
		close(release)

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Run = %v, want the shutdown deadline", err)
		}
		if err == nil || !strings.Contains(err.Error(), "still running 20s after shutdown began") {
			t.Errorf("Run = %v, want it to say servers are still running", err)
		}
	})
}

func TestWorker(t *testing.T) {
	t.Run("an error is a failure", func(t *testing.T) {
		boom := errors.New("boom")
		if err := run.Worker(func(context.Context) error { return boom }).Serve(); !errors.Is(err, boom) {
			t.Errorf("Serve = %v", err)
		}
	})
	t.Run("returning early is clean", func(t *testing.T) {
		if err := run.Worker(func(context.Context) error { return nil }).Serve(); err != nil {
			t.Errorf("Serve = %v", err)
		}
	})
	t.Run("context.Canceled it didn't cause is a failure", func(t *testing.T) {
		w := run.Worker(func(context.Context) error { return context.Canceled })
		if err := w.Serve(); !errors.Is(err, context.Canceled) {
			t.Errorf("Serve = %v", err)
		}
	})
	t.Run("shutdown before serve", func(t *testing.T) {
		called := false
		w := run.Worker(func(context.Context) error { called = true; return nil })
		if err := w.Shutdown(t.Context()); err != nil {
			t.Fatal(err)
		}
		if err := w.Serve(); err != nil || called {
			t.Errorf("Serve after Shutdown = %v, ran fn: %v", err, called)
		}
	})
}

func TestProbe(t *testing.T) {
	down := errors.New("dial tcp: connection refused")
	probe := run.NewProbe(map[string]run.Check{
		"mysql": func(context.Context) error { return down },
		"smtp":  func(context.Context) error { return nil },
		"cache": func(context.Context) error { return down },
	})

	rec := httptest.NewRecorder()
	probe.Ready().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("ready = %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	var problem map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"type":     "about:blank",
		"title":    "Service Unavailable",
		"status":   503.0,
		"detail":   "Failing checks: cache, mysql.",
		"instance": "/readyz",
	}
	for k, v := range want {
		if problem[k] != v {
			t.Errorf("problem[%s] = %v, want %v", k, problem[k], v)
		}
	}
	if strings.Contains(rec.Body.String(), "refused") {
		t.Errorf("the check error reached the response: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	probe.Live().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("live = %d", rec.Code)
	}
}

func TestRunServesHTTPUntilRequestsFinish(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		_, _ = io.WriteString(w, "done")
	})}
	shuttingDown := make(chan struct{})
	srv.RegisterOnShutdown(func() { close(shuttingDown) })
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	ran := make(chan error, 1)
	go func() {
		ran <- run.Run(ctx, run.Config{ShutdownTimeout: 10 * time.Second}, nil, discard, run.Listener(srv, ln))
	}()

	type result struct {
		body string
		err  error
	}
	got := make(chan result, 1)
	go func() {
		res, err := http.Get("http://" + ln.Addr().String())
		if err != nil {
			got <- result{err: err}
			return
		}
		defer res.Body.Close()
		b, err := io.ReadAll(res.Body)
		got <- result{string(b), err}
	}()

	<-entered
	cancel()
	<-shuttingDown
	close(release)
	if r := <-got; r.err != nil || r.body != "done" {
		t.Errorf("in-flight request = %q, %v", r.body, r.err)
	}
	if err := <-ran; err != nil {
		t.Errorf("Run = %v", err)
	}
}

func TestHTTP(t *testing.T) {
	t.Run("clean stop", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		srv := &http.Server{Addr: "127.0.0.1:0", Handler: http.NotFoundHandler()}
		if err := run.Run(ctx, run.Config{ShutdownTimeout: time.Second}, nil, discard, run.HTTP(srv)); err != nil {
			t.Errorf("Run = %v", err)
		}
	})
	t.Run("listen failure", func(t *testing.T) {
		srv := &http.Server{Addr: "127.0.0.1:-1"}
		err := run.Run(t.Context(), run.Config{ShutdownTimeout: time.Second}, nil, discard, run.HTTP(srv))
		if err == nil || !strings.Contains(err.Error(), "serve: http 127.0.0.1:-1") {
			t.Errorf("Run = %v", err)
		}
	})
	t.Run("shutdown timeout closes connections", func(t *testing.T) {
		entered := make(chan struct{})
		srv := &http.Server{Handler: http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			close(entered)
			<-r.Context().Done()
		})}
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		s := run.Listener(srv, ln)
		served := make(chan error, 1)
		go func() { served <- s.Serve() }()
		go func() {
			if res, err := http.Get("http://" + ln.Addr().String()); err == nil {
				res.Body.Close()
			}
		}()
		<-entered
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		if err := s.Shutdown(ctx); !errors.Is(err, context.Canceled) {
			t.Errorf("Shutdown = %v, want context.Canceled", err)
		}
		if err := <-served; err != nil {
			t.Errorf("Serve = %v", err)
		}
	})
}
