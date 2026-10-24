package config_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/caarlos0/env/v11"

	"github.com/vardius/go-api-boilerplate/internal/platform/config"
)

type listener struct {
	Addr    string        `env:"ADDR" envDefault:":8080" desc:"Address to listen on."`
	Timeout time.Duration `env:"TIMEOUT" envDefault:"5s" desc:"How long a request may take."`
}

type testConfig struct {
	Env      string        `env:"APP_ENV" envDefault:"production" desc:"production | development"`
	Issuer   string        `env:"ISSUER,required" desc:"Public URL of the auth API."`
	Name     string        `env:"NAME,notEmpty" desc:"Service name."`
	Password config.Secret `env:"PASSWORD" desc:"Database password."`
	HTTP     listener      `envPrefix:"HTTP_"`
	RPC      listener      `envPrefix:"RPC_"`
	Retries  int           `env:"RETRIES" envDefault:"3" desc:"Attempts."`
	Mail     struct {
		From string `env:"FROM" desc:"Sender address."`
	} `envPrefix:"MAIL_"`
	Skipped string `env:"-"`
}

func TestLoad(t *testing.T) {
	t.Setenv("ISSUER", "http://localhost:8080/api/auth")
	t.Setenv("NAME", "user")
	t.Setenv("PASSWORD", "hunter2")
	t.Setenv("RPC_ADDR", ":8081")
	t.Setenv("MAIL_FROM", "noreply@example.com")

	cfg, err := config.Load[testConfig]()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Issuer != "http://localhost:8080/api/auth" || cfg.Env != "production" || cfg.Password.Reveal() != "hunter2" {
		t.Errorf("cfg = %+v", cfg)
	}
	if cfg.HTTP.Addr != ":8080" || cfg.RPC.Addr != ":8081" || cfg.RPC.Timeout != 5*time.Second || cfg.Mail.From != "noreply@example.com" {
		t.Errorf("nested = %+v %+v %+v", cfg.HTTP, cfg.RPC, cfg.Mail)
	}
}

func TestLoadNamesEveryBadVariable(t *testing.T) {
	t.Setenv("NAME", "")
	t.Setenv("HTTP_TIMEOUT", "soon")
	t.Setenv("RETRIES", "many")

	cfg, err := config.Load[testConfig]()
	if err == nil {
		t.Fatal("Load succeeded")
	}
	if cfg != (testConfig{}) {
		t.Errorf("Load returned data with its error: %+v", cfg)
	}
	msg := err.Error()
	for _, want := range []string{
		"ISSUER is required",
		"NAME must not be empty",
		// Both listeners have a Timeout field, so a bad one could be either.
		`HTTP_TIMEOUT or RPC_TIMEOUT: unable to parse duration: time: invalid duration "soon"`,
		`RETRIES: strconv.ParseInt: parsing "many": invalid syntax`,
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q\nlacks %q", msg, want)
		}
	}
	if _, ok := errors.AsType[env.VarIsNotSetError](err); !ok {
		t.Error("the env error isn't reachable with errors.As")
	}
}

func TestSecretNeverPrints(t *testing.T) {
	s := config.Secret("hunter2")
	var logs bytes.Buffer
	slog.New(slog.NewJSONHandler(&logs, nil)).Info("x", "password", s)
	slog.New(slog.NewTextHandler(&logs, nil)).Info("x", "password", s)
	j, err := json.Marshal(struct{ P config.Secret }{s})
	if err != nil {
		t.Fatal(err)
	}
	text, err := s.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	outputs := []string{
		fmt.Sprint(s),
		fmt.Sprintf("%s %q %v %+v %#v %x %X", s, s, s, s, s, s, s),
		fmt.Sprintf("%d", s), // a verb that doesn't consult String
		fmt.Sprintf("%v", struct{ P config.Secret }{s}),
		s.String(),
		s.GoString(),
		string(j),
		string(text),
		logs.String(),
	}
	for _, out := range outputs {
		if strings.Contains(out, "hunter2") || strings.Contains(out, fmt.Sprintf("%x", "hunter2")) {
			t.Errorf("leaked: %s", out)
		}
		if !strings.Contains(out, "[redacted]") {
			t.Errorf("%q doesn't say [redacted]", out)
		}
	}
	if s.Reveal() != "hunter2" {
		t.Errorf("Reveal = %q", s.Reveal())
	}
}

func TestDescribe(t *testing.T) {
	got := config.Describe[testConfig]()
	want := []config.Var{
		{Name: "APP_ENV", Default: "production", Desc: "production | development"},
		{Name: "ISSUER", Desc: "Public URL of the auth API.", Required: true},
		{Name: "NAME", Desc: "Service name.", Required: true},
		{Name: "PASSWORD", Desc: "Database password.", Secret: true},
		{Name: "HTTP_ADDR", Default: ":8080", Desc: "Address to listen on."},
		{Name: "HTTP_TIMEOUT", Default: "5s", Desc: "How long a request may take."},
		{Name: "RPC_ADDR", Default: ":8080", Desc: "Address to listen on."},
		{Name: "RPC_TIMEOUT", Default: "5s", Desc: "How long a request may take."},
		{Name: "RETRIES", Default: "3", Desc: "Attempts."},
		{Name: "MAIL_FROM", Desc: "Sender address."},
	}
	if !slices.Equal(got, want) {
		t.Errorf("Describe =\n%+v\nwant\n%+v", got, want)
	}
}

func TestDescribeFollowsInitPointers(t *testing.T) {
	type db struct {
		URI config.Secret `env:"URI" desc:"Connection string."`
	}
	type cfg struct {
		Mongo *db `env:",init" envPrefix:"MONGODB_"`
		MySQL *db `envPrefix:"MYSQL_"`
		token string
	}
	_ = cfg{}.token
	got := config.Describe[cfg]()
	want := []config.Var{{Name: "MONGODB_URI", Desc: "Connection string.", Secret: true}}
	if !slices.Equal(got, want) {
		t.Errorf("Describe = %+v, want %+v", got, want)
	}
	if got := config.Describe[int](); len(got) != 0 {
		t.Errorf("Describe[int] = %+v", got)
	}
}

func TestMarkdown(t *testing.T) {
	got := config.Markdown([]config.Var{
		{Name: "APP_ENV", Default: "production", Desc: "production | development"},
		{Name: "ISSUER", Desc: "Public URL\nof the auth API.", Required: true},
		{Name: "SMTP_PASSWORD", Secret: true},
	})
	want := "| Variable | Default | Required | Description |\n" +
		"|---|---|---|---|\n" +
		"| `APP_ENV` | `production` |  | production \\| development |\n" +
		"| `ISSUER` |  | yes | Public URL of the auth API. |\n" +
		"| `SMTP_PASSWORD` |  |  | Secret. |\n"
	if got != want {
		t.Errorf("Markdown =\n%s\nwant\n%s", got, want)
	}
}
