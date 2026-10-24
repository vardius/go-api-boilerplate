package run

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
)

// HTTP serves srv on srv.Addr.
func HTTP(srv *http.Server) Server { return httpServer{srv: srv} }

// Listener serves srv on ln, which tests open on port 0 to learn the
// address.
func Listener(srv *http.Server, ln net.Listener) Server { return httpServer{srv: srv, ln: ln} }

type httpServer struct {
	srv *http.Server
	ln  net.Listener
}

func (s httpServer) addr() string {
	if s.ln != nil {
		return s.ln.Addr().String()
	}
	return s.srv.Addr
}

func (s httpServer) Serve() error {
	var err error
	if s.ln != nil {
		err = s.srv.Serve(s.ln)
	} else {
		err = s.srv.ListenAndServe()
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return fmt.Errorf("http %s: %w", s.addr(), err)
}

// Shutdown stops accepting connections and waits for requests in flight.
// When ctx ends first, it closes the connections that are left.
func (s httpServer) Shutdown(ctx context.Context) error {
	if err := s.srv.Shutdown(ctx); err != nil {
		return errors.Join(fmt.Errorf("http %s: %w", s.addr(), err), s.srv.Close())
	}
	return nil
}

// Worker runs fn as a server. Shutdown cancels fn's context and waits for fn
// to return; fn returning the context's error then counts as a clean stop.
// A Worker serves once.
func Worker(fn func(ctx context.Context) error) Server { return &worker{fn: fn} }

type worker struct {
	fn func(ctx context.Context) error

	mu      sync.Mutex
	stopped bool
	cancel  context.CancelFunc // set once Serve starts
	done    chan struct{}      // closed when Serve returns
}

func (w *worker) Serve() error {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	w.cancel, w.done = cancel, done
	w.mu.Unlock()
	defer close(done)
	defer cancel()

	err := w.fn(ctx)
	if ctx.Err() != nil && errors.Is(err, context.Canceled) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("worker: %w", err)
	}
	return nil
}

func (w *worker) Shutdown(ctx context.Context) error {
	w.mu.Lock()
	w.stopped = true
	cancel, done := w.cancel, w.done
	w.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("worker: %w", ctx.Err())
	}
}
