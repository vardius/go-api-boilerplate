package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/platform/config"
)

func names(vars []config.Var) []string {
	out := make([]string, len(vars))
	for i, v := range vars {
		out[i] = v.Name
	}
	return out
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// composeEnv returns the variables a compose file sets for the auth
// service.
func composeEnv(t *testing.T, path string) []string {
	var out []string
	inAuth, inEnv := false, false
	for line := range strings.Lines(read(t, path)) {
		line = strings.TrimRight(line, "\n")
		indent := len(line) - len(strings.TrimLeft(line, " "))
		switch {
		case strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#"):
		case indent == 2:
			inAuth, inEnv = line == "  auth:", false
		case inAuth && indent == 4:
			inEnv = line == "    environment:"
		case inAuth && inEnv && indent == 6:
			name, _, _ := strings.Cut(strings.TrimSpace(line), ":")
			out = append(out, name)
		}
	}
	return out
}

var helmName = regexp.MustCompile(`(?m)^\s*- name: ([A-Z][A-Z0-9_]*)$`)

// helmEnv returns the variables the chart sets for the auth service and
// for every service.
func helmEnv(t *testing.T) []string {
	const chart = "../../deploy/helm/go-api-boilerplate/templates/"
	helpers := read(t, chart+"_helpers.tpl")
	shared := helpers[strings.Index(helpers, `define "gab.env"`):]
	var out []string
	for _, src := range []string{read(t, chart+"auth.yaml"), shared} {
		for _, m := range helmName.FindAllStringSubmatch(src, -1) {
			out = append(out, m[1])
		}
	}
	return out
}

// The variables compose and the chart set are the ones the config reads,
// so none is silently ignored. Store variables are checked by the build
// that has that store.
func TestConfigReadsTheDeployedVariables(t *testing.T) {
	read := names(config.Describe[Config]())
	deployed := slices.Concat(composeEnv(t, "../../compose.yaml"), composeEnv(t, "../../compose.mysql.yaml"), composeEnv(t, "../../compose.mongodb.yaml"), helmEnv(t))
	if len(deployed) < 10 {
		t.Fatalf("found only %v; has the layout of the deploy files changed?", deployed)
	}
	for _, name := range deployed {
		store, _, _ := strings.Cut(name, "_")
		if (store == "MYSQL" || store == "MONGODB") && !slices.ContainsFunc(read, func(n string) bool { return strings.HasPrefix(n, store+"_") }) {
			continue
		}
		if !slices.Contains(read, name) {
			t.Errorf("%s is deployed but the config doesn't read it", name)
		}
	}
}

func TestEveryVariableIsDescribed(t *testing.T) {
	for _, v := range config.Describe[Config]() {
		if v.Desc == "" {
			t.Errorf("%s has no desc", v.Name)
		}
	}
	md := config.Markdown(config.Describe[Config]())
	for _, name := range []string{"ISSUER", "WEB_URL", "USER_RPC_URL", "SIGNING_KEY_FILE", "PREVIOUS_KEYS_DIR", "COOKIE_SECURE", "REFRESH_COOKIE_PATH", "TRUSTED_PROXIES", "MIGRATE_ON_START", "RATE_LIMIT_RATE", "HTTP_ADDR", "LOG_LEVEL", "SHUTDOWN_DELAY", "AUDIENCE", "APP_ENV"} {
		if !strings.Contains(md, "`"+name+"`") {
			t.Errorf("the reference doesn't list %s", name)
		}
	}
}

// The migration job sets only the store's variables and the log's.
func TestMigrateReadsOnlyTheStore(t *testing.T) {
	for _, name := range names(config.Describe[migrateConfig]()) {
		if !strings.HasPrefix(name, "LOG_") && !strings.HasPrefix(name, "MYSQL_") && !strings.HasPrefix(name, "MONGODB_") {
			t.Errorf("migrate reads %s", name)
		}
	}
}

func listen(t *testing.T, h http.Handler) string {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	_, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	return port
}

func TestHealthcheck(t *testing.T) {
	ready := true
	port := listen(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readyz" || !ready {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	t.Setenv("HTTP_ADDR", ":"+port)
	if err := healthcheck(t.Context()); err != nil {
		t.Fatalf("ready: %v", err)
	}
	ready = false
	if err := healthcheck(t.Context()); err == nil {
		t.Fatal("not ready, yet healthy")
	}
	t.Setenv("HTTP_ADDR", "no-port")
	if err := healthcheck(t.Context()); err == nil {
		t.Fatal("no port, yet healthy")
	}
}

// needsNoDatabase skips a test of serve in a build with a database store.
func needsNoDatabase(t *testing.T) {
	if reflect.TypeFor[storeConfig]().NumField() > 0 {
		t.Skip("this build's store needs a database")
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

// serve starts with what compose sets and answers on its routes, then
// stops when told to.
func TestServe(t *testing.T) {
	needsNoDatabase(t)
	port := freePort(t)
	for k, v := range map[string]string{
		"APP_ENV":          "development",
		"HTTP_ADDR":        "127.0.0.1:" + port,
		"ISSUER":           "http://localhost:8080/api/auth",
		"WEB_URL":          "http://localhost:8080",
		"USER_RPC_URL":     "http://127.0.0.1:1",
		"COOKIE_SECURE":    "false",
		"SHUTDOWN_DELAY":   "0s",
		"MIGRATE_ON_START": "true",
		"LOG_LEVEL":        "error",
	} {
		t.Setenv(k, v)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- serve(ctx) }()

	base := "http://127.0.0.1:" + port
	deadline := time.Now().Add(10 * time.Second)
	for {
		if err := healthcheck(t.Context()); err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("not ready in time: %v", <-done)
		}
		time.Sleep(50 * time.Millisecond)
	}
	for path, want := range map[string]string{
		"/.well-known/jwks.json":                  `"kty":"EC"`,
		"/.well-known/oauth-authorization-server": `"issuer":"http://localhost:8080/api/auth"`,
	} {
		res, err := http.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusOK || !strings.Contains(string(body), want) {
			t.Errorf("GET %s = %d %s", path, res.StatusCode, body)
		}
	}

	cancel()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("serve stopped with %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("serve didn't stop")
	}
}

func TestServeRefusesAMissingKeyOutsideDevelopment(t *testing.T) {
	needsNoDatabase(t)
	for k, v := range map[string]string{
		"APP_ENV":      "production",
		"HTTP_ADDR":    "127.0.0.1:" + freePort(t),
		"ISSUER":       "http://localhost:8080/api/auth",
		"WEB_URL":      "http://localhost:8080",
		"USER_RPC_URL": "http://127.0.0.1:1",
		"LOG_LEVEL":    "error",
	} {
		t.Setenv(k, v)
	}
	if err := serve(t.Context()); err == nil || !strings.Contains(fmt.Sprint(err), "SIGNING_KEY_FILE") {
		t.Fatalf("serve = %v, want a missing key error", err)
	}
}
