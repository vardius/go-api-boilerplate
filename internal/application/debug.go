package application

import (
	"context"
	_ "expvar" // Register the expvar handlers
	"net/http"
	_ "net/http/pprof" // Register the pprof handlers
)

// DebugAdapter ./...
type DebugAdapter struct {
	address string
	*http.ServeMux
}

// NewDebugAdapter provides new debug adapter
// /debug/pprof - Added to the default mux by importing the net/http/pprof package.
// /debug/vars - Added to the default mux by importing the expvar package.
func NewDebugAdapter(address string) *DebugAdapter {
	return &DebugAdapter{
		address:  address,
		ServeMux: http.DefaultServeMux,
	}
}

// Start starts http server
func (adapter *DebugAdapter) Start(ctx context.Context) error {
	return http.ListenAndServe(adapter.address, adapter)
}

// Stop stops debug server
// does nothing always returns nil
func (adapter *DebugAdapter) Stop(ctx context.Context) error {
	return nil
}
