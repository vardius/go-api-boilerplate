package config

import (
	"runtime"
	"sync"

	"github.com/caarlos0/env"
)

var (
	// Env stores environment values
	Env  *environment
	once sync.Once
)

type environment struct {
	Environment string   `env:"ENV"     envDefault:"development"`
	Secret      string   `env:"SECRET"  envDefault:"secret"`
	Origins     []string `env:"ORIGINS" envSeparator:"|"` // Origins should follow format: scheme "://" host [ ":" port ]

	ClientID     string `env:"CLIENT_ID"     envDefault:"clientId"`
	ClientSecret string `env:"CLIENT_SECRET" envDefault:"clientSecret"`

	Host     string `env:"HOST"      envDefault:"0.0.0.0"`
	PortHTTP int    `env:"PORT_HTTP" envDefault:"3000"`
	PortGRPC int    `env:"PORT_GRPC" envDefault:"3001"`

	DbHost string `env:"DB_HOST" envDefault:"0.0.0.0"`
	DbPort int    `env:"DB_PORT" envDefault:"3306"`
	DbUser string `env:"DB_USER" envDefault:"root"`
	DbPass string `env:"DB_PASS" envDefault:"password"`
	DbName string `env:"DB_NAME" envDefault:"goapiboilerplate"`

	AuthHost string `env:"AUTH_HOST" envDefault:"0.0.0.0"`

	CommandBusQueueSize int `env:"COMMAND_BUS_BUFFER" envDefault:"0"`
	EventBusQueueSize   int `env:"EVENT_BUS_BUFFER" envDefault:"0"`
}

func init() {
	once.Do(func() {
		env.Parse(Env)

		if Env.CommandBusQueueSize == 0 {
			Env.CommandBusQueueSize = runtime.NumCPU()
		}
		if Env.EventBusQueueSize == 0 {
			Env.EventBusQueueSize = runtime.NumCPU()
		}
	})
}
