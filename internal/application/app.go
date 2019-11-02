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
	stop := func() {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, app.shutdownTimeout)
		defer cancel()

		app.logger.Info(ctxWithTimeout, "shutting down...\n")

		var errorOccurred bool
		errCh := make(chan error, len(app.adapters))

		for _, adapter := range app.adapters {
			go func(adapter Adapter) {
				errCh <- adapter.Stop(ctxWithTimeout)
			}(adapter)
		}

		for i := 0; i < len(app.adapters); i++ {
			if err := <-errCh; err != nil {
				errorOccurred = true
				app.logger.Critical(ctxWithTimeout, "shutdown error: %v\n", err)
			}
		}

		app.logger.Info(ctxWithTimeout, "gracefully stopped\n")

		if errorOccurred {
			os.Exit(1)
		}
	}

	for _, adapter := range app.adapters {
		go func(adapter Adapter) {
			app.logger.Critical(ctx, "adapter start error: %v\n", adapter.Start(ctx))
		}(adapter)
	}

	shutdown.GracefulStop(stop)
}
