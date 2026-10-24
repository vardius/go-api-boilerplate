package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/platform/config"
)

// healthcheck asks this host's server whether it is ready, for container
// runtimes whose image has no curl.
func healthcheck(ctx context.Context) error {
	cfg, err := config.Load[healthcheckConfig]()
	if err != nil {
		return err
	}
	_, port, err := net.SplitHostPort(cfg.HTTP.Addr)
	if err != nil {
		return fmt.Errorf("HTTP_ADDR %q: %w", cfg.HTTP.Addr, err)
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+net.JoinHostPort("127.0.0.1", port)+"/readyz", nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return errors.New("not ready: " + res.Status)
	}
	return nil
}
