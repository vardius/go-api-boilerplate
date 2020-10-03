package application

import (
	"context"
	"os"
	"time"

	"github.com/vardius/golog"
	"github.com/vardius/shutdown"
)

// Adapter interface
type Adapter interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// App represents application service
type App struct {
	adapters        []Adapter
	shutdownTimeout time.Duration

	logger golog.Logger
}

// New provides new service application
func New(logger golog.Logger) *App {
	return &App{
		shutdownTimeout: 5 * time.Second, // Default shutdown timeout
		logger:          logger,
	}
}

// AddAdapters adds adapters to application service
func (app *App) AddAdapters(adapters ...Adapter) {
	app.adapters = append(app.adapters, adapters...)
}

// WithShutdownTimeout overrides default shutdown timout
func (app *App) WithShutdownTimeout(timeout time.Duration) {
	app.shutdownTimeout = timeout
}

// Run runs the service application
func (app *App) Run(ctx context.Context) {
	for _, adapter := range app.adapters {
		go func(adapter Adapter) {
			if err := adapter.Start(ctx); err != nil {
				app.logger.Critical(ctx, "adapter start error: %v", adapter.Start(ctx))
				os.Exit(1)
			}
		}(adapter)
	}

	shutdown.GracefulStop(func() { app.stop(ctx) })
}

func (app *App) stop(ctx context.Context) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, app.shutdownTimeout)
	defer cancel()

	app.logger.Info(ctxWithTimeout, "shutting down...")

	errCh := make(chan error, len(app.adapters))

	for _, adapter := range app.adapters {
		go func(adapter Adapter) {
			errCh <- adapter.Stop(ctxWithTimeout)
		}(adapter)
	}

	for i := 0; i < len(app.adapters); i++ {
		if err := <-errCh; err != nil {
			// calling Goexit terminates that goroutine without returning (previous defers would not run)
			go func(err error) {
				app.logger.Critical(ctxWithTimeout, "shutdown error: %v", err)
				os.Exit(1)
			}(err)
			return
		}
	}

	app.logger.Info(ctxWithTimeout, "gracefully stopped")
}
