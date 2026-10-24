// Package run starts a process's servers and stops them cleanly.
//
// Run serves until the process gets SIGINT or SIGTERM, its context ends, or
// a server fails. Then it drains: readiness turns 503 so load balancers stop
// sending traffic, it waits ShutdownDelay for them to notice, and it shuts
// every server down at once, giving them ShutdownTimeout to finish what
// they are doing. It returns what failed; main decides the exit code.
package run

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Config times the shutdown. Services embed it without a prefix.
type Config struct {
	ShutdownDelay   time.Duration `env:"SHUTDOWN_DELAY" envDefault:"5s" desc:"How long readiness reports 503 before servers stop, so load balancers stop routing here first."`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"20s" desc:"How long servers get to finish in-flight work once they stop."`
}

// Server is something Run starts and stops: an HTTP listener or a
// background worker.
type Server interface {
	// Serve blocks until the server stops. It returns nil after a clean
	// Shutdown.
	Serve() error
	// Shutdown stops the server, waiting for in-flight work until ctx ends.
	Shutdown(ctx context.Context) error
}

// Run serves servers until ctx ends, the process is told to stop, or a
// server's Serve fails, and then drains probe and shuts every server down.
// It returns the joined errors of the servers that failed, nil after a
// clean stop.
func Run(ctx context.Context, cfg Config, probe *Probe, log *slog.Logger, servers ...Server) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	var (
		mu   sync.Mutex
		errs []error
		wg   sync.WaitGroup
	)
	fail := func(err error) {
		mu.Lock()
		errs = append(errs, err)
		mu.Unlock()
		cancel(err)
	}
	for _, s := range servers {
		wg.Go(func() {
			if err := s.Serve(); err != nil {
				fail(fmt.Errorf("serve: %w", err))
			}
		})
	}

	<-ctx.Done()
	// A second signal now kills the process instead of waiting for the drain.
	stop()
	log.Info("draining", "reason", context.Cause(ctx).Error(), "delay", cfg.ShutdownDelay)
	probe.drain()
	time.Sleep(cfg.ShutdownDelay)

	log.Info("shutting down", "timeout", cfg.ShutdownTimeout)
	shutdownCtx, cancelShutdown := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancelShutdown()
	var shutdowns sync.WaitGroup
	for _, s := range servers {
		shutdowns.Go(func() {
			if err := s.Shutdown(shutdownCtx); err != nil {
				fail(fmt.Errorf("shutdown: %w", err))
			}
		})
	}
	shutdowns.Wait()

	served := make(chan struct{})
	go func() {
		wg.Wait()
		close(served)
	}()
	select {
	case <-served:
	case <-shutdownCtx.Done():
		fail(fmt.Errorf("servers still running %s after shutdown began", cfg.ShutdownTimeout))
	}

	mu.Lock()
	defer mu.Unlock()
	return errors.Join(errs...)
}
