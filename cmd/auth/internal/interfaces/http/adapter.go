package http

import (
	"context"
	"net"
	"net/http"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
)

// Adapter is http server app adapter
type Adapter struct {
	*http.Server
}

// NewAdapter provides new primary adapter
func NewAdapter(address string, router http.Handler) *Adapter {
	return &Adapter{&http.Server{
		Addr:         address,
		ReadTimeout:  config.Env.HTTP.ReadTimeout,
		WriteTimeout: config.Env.HTTP.WriteTimeout,
		IdleTimeout:  config.Env.HTTP.IdleTimeout, // limits server-side the amount of time a Keep-Alive connection will be kept idle before being reused
		Handler:      router,
	},
	}
}

// Start start http application adapter
func (adapter *Adapter) Start(ctx context.Context) error {
	adapter.BaseContext = func(_ net.Listener) context.Context { return ctx }

	if err := adapter.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Stop stops http application adapter
func (adapter *Adapter) Stop(ctx context.Context) error {
	return adapter.Shutdown(ctx)
}
