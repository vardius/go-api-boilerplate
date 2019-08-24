package http

import (
	"context"
	"net/http"
	"time"
	"github.com/vardius/go-api-boilerplate/cmd/user/application"
)

type httpAdapter struct {
	address string
	server  *http.Server
	router  http.Handler
}

// NewAdapter provides new primary adapter
func NewAdapter(address string, router http.Handler) application.Adapter {
	return &httpAdapter{
		router:  router,
		address: address,
	}
}

// Start starts http server
func (adapter *httpAdapter) Start(ctx context.Context) error {
	// TODO: Allow to configure with env vars
	adapter.server = &http.Server{
		Addr:         adapter.address,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      adapter.router,
	}

	return adapter.server.ListenAndServe()
}

// Stop stops http server
func (adapter *httpAdapter) Stop(ctx context.Context) error {
	return adapter.server.Shutdown(ctx)
}
