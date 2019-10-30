package http

import (
	"context"
	"net/http"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
)

type HttpAdapter struct {
	*http.Server
}

// NewAdapter provides new primary adapter
func NewAdapter(address string, router http.Handler) *HttpAdapter {
	return &HttpAdapter{&http.Server{
		Addr:         address,
		ReadTimeout:  config.Env.HTTP.ReadTimeout,
		WriteTimeout: config.Env.HTTP.WriteTimeout,
		IdleTimeout:  config.Env.HTTP.IdleTimeout, // limits server-side the amount of time a Keep-Alive connection will be kept idle before being reused
		Handler:      router,
	},
	}
}

// Start starts http server
func (adapter *HttpAdapter) Start(ctx context.Context) error {
	return adapter.ListenAndServe()
}

// Stop stops http server
func (adapter *HttpAdapter) Stop(ctx context.Context) error {
	return adapter.Shutdown(ctx)
}
