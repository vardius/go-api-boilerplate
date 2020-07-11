package application

import (
	"context"
	_ "expvar" // Register the expvar handlers
	"net/http"
	_ "net/http/pprof" // Register the pprof handlers
)

// DebugAdapter ./...
type DebugAdapter struct {
	*http.Server
}

// NewDebugAdapter provides new debug adapter
// /debug/pprof - Added to the default mux by importing the net/http/pprof package.
// /debug/vars - Added to the default mux by importing the expvar package.
func NewDebugAdapter(address string) *DebugAdapter {
	return &DebugAdapter{
		&http.Server{
			Addr:    address,
			Handler: http.DefaultServeMux,
		},
	}
}

// Start start http application adapter
func (adapter *DebugAdapter) Start(ctx context.Context) error {
	if err := adapter.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Stop stops http application adapter
func (adapter *DebugAdapter) Stop(ctx context.Context) error {
	return adapter.Shutdown(ctx)
}
