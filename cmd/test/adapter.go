package main

import (
	"context"
	"log"
	"net/http"

	"github.com/vardius/go-api-boilerplate/cmd/test/config"
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

// Start starts http server
func (adapter *Adapter) Start(ctx context.Context) error {
	log.Printf("Adapter start %s\n", adapter.Addr)
	return adapter.ListenAndServe()
}

// Stop stops http server
func (adapter *Adapter) Stop(ctx context.Context) error {
	log.Printf("Adapter stop\n")
	return adapter.Shutdown(ctx)
}
