package http

import (
	"context"
	"net"
	"net/http"
)

// Adapter is http server app adapter
type Adapter struct {
	httpServer *http.Server
}

// NewAdapter provides new primary HTTP adapter
func NewAdapter(httpServer *http.Server) *Adapter {
	return &Adapter{
		httpServer: httpServer,
	}
}

// Start start http application adapter
func (adapter *Adapter) Start(ctx context.Context) error {
	adapter.httpServer.BaseContext = func(_ net.Listener) context.Context { return ctx }

	if err := adapter.httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Stop stops http application adapter
func (adapter *Adapter) Stop(ctx context.Context) error {
	return adapter.httpServer.Shutdown(ctx)
}
