package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/platform/config"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
)

// healthConfig is all healthcheck reads, so it runs in a container whatever
// else the environment lacks.
type healthConfig struct {
	HTTP httpx.ServerConfig `envPrefix:"HTTP_"`
}

// healthTimeout stays under the compose healthcheck's 3s timeout.
const healthTimeout = 2 * time.Second

// healthcheck asks the service on this host whether it is ready, for a
// container image with no shell or curl.
func healthcheck(ctx context.Context) error {
	cfg, err := config.Load[healthConfig]()
	if err != nil {
		return err
	}
	_, port, err := net.SplitHostPort(cfg.HTTP.Addr)
	if err != nil {
		return fmt.Errorf("HTTP_ADDR %q: %w", cfg.HTTP.Addr, err)
	}
	ctx, cancel := context.WithTimeout(ctx, healthTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+net.JoinHostPort("127.0.0.1", port)+"/readyz", nil)
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: /readyz answered %s", res.Status)
	}
	return nil
}
