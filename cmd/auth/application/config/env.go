package config

import (
	"runtime"

	"github.com/caarlos0/env/v6"
)

// Env stores environment values
var Env *environment

type environment struct {
	Environment string   `env:"ENV"     envDefault:"development"`
	Secret      string   `env:"SECRET"  envDefault:"secret"`
	Origins     []string `env:"ORIGINS" envSeparator:"|"` // Origins should follow format: scheme "://" host [ ":" port ]

	Host     string `env:"HOST"      envDefault:"0.0.0.0"`
	PortHTTP int    `env:"PORT_HTTP" envDefault:"3000"`
	PortGRPC int    `env:"PORT_GRPC" envDefault:"3001"`

	DbHost string `env:"DB_HOST" envDefault:"0.0.0.0"`
	DbPort int    `env:"DB_PORT" envDefault:"3306"`
	DbUser string `env:"DB_USER" envDefault:"root"`
	DbPass string `env:"DB_PASS" envDefault:"password"`
	DbName string `env:"DB_NAME" envDefault:"goapiboilerplate"`

	UserHost         string `env:"USER_HOST"          envDefault:"0.0.0.0"`
	UserClientID     string `env:"USER_CLIENT_ID"     envDefault:"clientId"`
	UserClientSecret string `env:"USER_CLIENT_SECRET" envDefault:"clientSecret"`

	PubSubHost string `env:"PUBSUB_HOST" envDefault:"0.0.0.0"`

	CommandBusQueueSize int `env:"COMMAND_BUS_BUFFER" envDefault:"0"`
}

func init() {
	Env = &environment{}
	env.Parse(Env)

	if Env.CommandBusQueueSize == 0 {
		Env.CommandBusQueueSize = runtime.NumCPU()
	}
}
