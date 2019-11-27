package config

import (
	"log"
	"time"

	"github.com/caarlos0/env/v6"
)

// Env stores environment values
var Env *environment

type environment struct {
	App struct {
		Environment     string        `env:"ENV"     envDefault:"development"`
		ShutdownTimeout time.Duration `env:"HTTP_SERVER_SHUTDOWN_TIMEOUT" envDefault:"5s"`
		Secret          string        `env:"SECRET"  envDefault:"secret"`
	}
	Debug struct {
		Host string `env:"DEBUG_HOST"      envDefault:"0.0.0.0"`
		Port int    `env:"DEBUG_PORT_HTTP" envDefault:"4000"`
	}
	HTTP struct {
		Origins []string `env:"ORIGINS" envSeparator:"|"` // Origins should follow format: scheme "://" host [ ":" port ]

		Host string `env:"HOST"      envDefault:"0.0.0.0"`
		Port int    `env:"PORT_HTTP" envDefault:"3000"`

		ReadTimeout  time.Duration `env:"HTTP_SERVER_READ_TIMEOUT" envDefault:"5s"`
		WriteTimeout time.Duration `env:"HTTP_SERVER_WRITE_TIMEOUT" envDefault:"10s"`
		IdleTimeout  time.Duration `env:"HTTP_SERVER_SHUTDOWN_TIMEOUT" envDefault:"120s"`
	}
}

func init() {
	Env = &environment{}

	env.Parse(&Env.App)
	env.Parse(&Env.Debug)
	env.Parse(&Env.HTTP)

	log.Printf("Env:\n%v\n", Env)
}
