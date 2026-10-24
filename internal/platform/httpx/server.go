// Package httpx is the HTTP kit the services share: a server with every
// timeout set, handlers that return errors, one table that turns errors
// into RFC 9457 problems, strict JSON decoding, paging, the client IP behind
// proxies, and the middleware chain.
//
// A service wraps its router, outermost first, in
//
//	httpx.Chain(
//		httpx.Recover(log),
//		httpx.RequestID,
//		httpx.Trace,
//		httpx.Log(log),
//		httpx.SecurityHeaders,
//		httpx.RateLimit(cfg.RateLimit, cfg.TrustedProxies),
//		httpx.BodyLimit(1<<20),
//		httpx.CrossOrigin,
//	)
//
// The router sets r.Pattern on the request it is given, and Trace and Log
// read it after the router returns to name the route. A middleware between
// them that hands on a new request, to add to its context, does so with
// ServeWithContext, which copies the pattern back.
package httpx

import (
	"net/http"
	"time"
)

// ServerConfig sets an HTTP server's address and limits. Services nest it
// with envPrefix:"HTTP_". Zero fields take the defaults shown.
type ServerConfig struct {
	Addr              string        `env:"ADDR" envDefault:":8080" desc:"Address the server listens on."`
	ReadHeaderTimeout time.Duration `env:"READ_HEADER_TIMEOUT" envDefault:"5s" desc:"How long a client may take to send the request headers."`
	ReadTimeout       time.Duration `env:"READ_TIMEOUT" envDefault:"15s" desc:"How long a client may take to send the whole request."`
	WriteTimeout      time.Duration `env:"WRITE_TIMEOUT" envDefault:"15s" desc:"How long writing the response may take, from the end of the request headers."`
	IdleTimeout       time.Duration `env:"IDLE_TIMEOUT" envDefault:"60s" desc:"How long a keep-alive connection may wait for its next request."`
	MaxHeaderBytes    int           `env:"MAX_HEADER_BYTES" envDefault:"65536" desc:"Largest request header block accepted, in bytes."`
}

func (c ServerConfig) withDefaults() ServerConfig {
	if c.Addr == "" {
		c.Addr = ":8080"
	}
	if c.ReadHeaderTimeout <= 0 {
		c.ReadHeaderTimeout = 5 * time.Second
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = 15 * time.Second
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = 15 * time.Second
	}
	if c.IdleTimeout <= 0 {
		c.IdleTimeout = 60 * time.Second
	}
	if c.MaxHeaderBytes <= 0 {
		c.MaxHeaderBytes = 64 << 10
	}
	return c
}

// NewServer returns a server for h with cfg's address and limits.
func NewServer(cfg ServerConfig, h http.Handler) *http.Server {
	cfg = cfg.withDefaults()
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           h,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}
}
