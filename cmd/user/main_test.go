package main

import (
	"context"
	"io"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/platform/config"
)

// composeEnv is what compose.yaml sets for the user service.
var composeEnv = map[string]string{
	"LOG_FORMAT":           "text",
	"LOG_LEVEL":            "info",
	"WEB_URL":              "http://localhost:8080",
	"AUTH_ISSUER":          "http://localhost:8080/api/auth",
	"AUTH_JWKS_URL":        "http://auth:8080/.well-known/jwks.json",
	"SMTP_ADDR":            "mailpit:1025",
	"SMTP_FROM":            "go-api-boilerplate <no-reply@go-api-boilerplate.localhost>",
	"SMTP_ALLOW_INSECURE":  "true",
	"MAIL_PREVIEW_URL":     "http://localhost:8025",
	"GOOGLE_CLIENT_ID":     "",
	"GOOGLE_CLIENT_SECRET": "",
	"GOOGLE_REDIRECT_URL":  "http://localhost:8080/api/user/v1/google/callback",
	"TRUSTED_PROXIES":      "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16",
}

func setEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for k, v := range env {
		t.Setenv(k, v)
	}
}

func TestComposeEnvironmentLoads(t *testing.T) {
	setEnv(t, composeEnv)
	cfg, err := config.Load[Config]()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WebURL.String() != "http://localhost:8080" || !cfg.SMTP.AllowInsecure || cfg.Google.Enabled() || len(cfg.TrustedProxies) != 3 {
		t.Errorf("loaded %+v", cfg)
	}
	if cfg.HTTP.Addr != ":8080" || cfg.RPCAddr != ":8081" {
		t.Errorf("listening on %s and %s, want :8080 and :8081 as the chart's ports", cfg.HTTP.Addr, cfg.RPCAddr)
	}
}

// Every variable compose and the Helm chart set for the user service is one
// the service reads.
func TestDeployVariablesAreRead(t *testing.T) {
	var names []string
	for _, v := range config.Describe[Config]() {
		names = append(names, v.Name)
	}
	helm := []string{
		"LOG_LEVEL", "MIGRATE_ON_START", "WEB_URL", "AUTH_ISSUER", "AUTH_JWKS_URL", "TRUSTED_PROXIES",
		"SMTP_ADDR", "SMTP_FROM", "SMTP_USERNAME", "SMTP_PASSWORD", "GOOGLE_CLIENT_ID", "GOOGLE_REDIRECT_URL", "GOOGLE_CLIENT_SECRET",
	}
	for _, name := range slices.Concat(slices.Collect(maps.Keys(composeEnv)), helm) {
		if !slices.Contains(names, name) {
			t.Errorf("%s is set by the deployment but not read", name)
		}
	}
}

func TestServeRefusesABadWebURL(t *testing.T) {
	setEnv(t, composeEnv)
	t.Setenv("WEB_URL", "/relative")
	if err := serve(t.Context()); err == nil || !strings.Contains(err.Error(), "WEB_URL") {
		t.Fatalf("serve = %v, want a WEB_URL error", err)
	}
}

func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	return port
}

// serve starts with compose's environment and answers on both listeners.
func TestServe(t *testing.T) {
	setEnv(t, composeEnv)
	httpPort, rpcPort := freePort(t), freePort(t)
	t.Setenv("HTTP_ADDR", "127.0.0.1:"+httpPort)
	t.Setenv("RPC_ADDR", "127.0.0.1:"+rpcPort)
	t.Setenv("SHUTDOWN_DELAY", "0s")
	t.Setenv("LOG_LEVEL", "error")

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- serve(ctx) }()

	wait, stop := context.WithTimeout(t.Context(), 10*time.Second)
	defer stop()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for healthcheck(wait) != nil {
		select {
		case <-wait.Done():
			t.Fatal("the service never became ready")
		case <-tick.C:
		}
	}
	res, err := http.Get("http://127.0.0.1:" + httpPort + "/v1/config")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(body), `"mail_preview_url":"http://localhost:8025"`) {
		t.Errorf("config = %s", body)
	}
	res, err = http.Post("http://127.0.0.1:"+rpcPort+"/user.v1.LoginService/RedeemLoginCode", "application/json", strings.NewReader(`{"code":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("RPC without a token = %d, want 401", res.StatusCode)
	}

	// Shutdown waits up to 5s on a connection the client dialed and never
	// used, so the client lets go of its spares first.
	http.DefaultClient.CloseIdleConnections()
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("serve = %v", err)
	}
}

func TestHealthcheck(t *testing.T) {
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readyz" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	_, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	t.Setenv("HTTP_ADDR", ":"+port)

	if err := healthcheck(t.Context()); err != nil {
		t.Errorf("healthcheck of a ready service = %v", err)
	}
	status = http.StatusServiceUnavailable
	if err := healthcheck(t.Context()); err == nil {
		t.Error("healthcheck passed a service that isn't ready")
	}
	t.Setenv("HTTP_ADDR", "no-port")
	if err := healthcheck(t.Context()); err == nil {
		t.Error("healthcheck passed with a bad HTTP_ADDR")
	}
}

// The memory build has nothing to migrate, and migrate reads no other
// configuration, as the chart's migration Job sets only the store's.
func TestMigrate(t *testing.T) {
	if err := migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
}
