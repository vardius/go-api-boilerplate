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
	logger   golog.Logger
	adapters []Adapter
}

// New provides new service application
func New(logger golog.Logger) *App {
	return &App{
		logger: logger,
	}
}

// AddAdapters adds adapters to application service
func (app *App) AddAdapters(adapters ...Adapter) *App {
	return &App{
		adapters: append(app.adapters, adapters...),
	}
}

// Run runs the service application
func (app *App) Run(ctx context.Context) {
	stop := func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		app.logger.Info(ctx, "shutting down...\n")

		for _, adapter := range app.adapters {
			go func(adapter Adapter) {
				if err := adapter.Stop(ctx); err != nil {
					app.logger.Critical(ctx, "shutdown error: %v\n", err)
					os.Exit(1)
				}
			}(adapter)
		}

		app.logger.Info(ctx, "gracefully stopped\n")
	}

	for _, adapter := range app.adapters {
		go func(adapter Adapter) {
			err := adapter.Start(ctx)

			stop()

			if err != nil {
				app.logger.Critical(ctx, "%v\n", adapter.Start(ctx))
				os.Exit(1)
			}
		}(adapter)
	}

	shutdown.GracefulStop(stop)
}
