package config

import (
	"log"
	"runtime"
	"time"

	"github.com/caarlos0/env/v6"
)

// Env stores environment values
var Env *environment

type environment struct {
	APP struct {
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
	GRPC struct {
		Host string `env:"HOST"      envDefault:"0.0.0.0"`
		Port int    `env:"PORT_GRPC" envDefault:"3001"`

		ServerMinTime time.Duration `env:"GRPC_SERVER_MIN_TIME" envDefault:"5m"` // if a client pings more than once every 5 minutes (default), terminate the connection
		ServerTime    time.Duration `env:"GRPC_SERVER_TIME" envDefault:"2h"`     // ping the client if it is idle for 2 hours (default) to ensure the connection is still active
		ServerTimeout time.Duration `env:"GRPC_SERVER_TIMEOUT" envDefault:"20s"` // wait 20 second (default) for the ping ack before assuming the connection is dead
		ConnTime      time.Duration `env:"GRPC_CONN_TIME" envDefault:"10s"`      // send pings every 10 seconds if there is no activity
		ConnTimeout   time.Duration `env:"GRPC_CONN_TIMEOUT" envDefault:"20s"`   // wait 20 second for ping ack before considering the connection dead
	}
	DB struct {
		Host     string `env:"DB_HOST" envDefault:"0.0.0.0"`
		Port     int    `env:"DB_PORT" envDefault:"3306"`
		User     string `env:"DB_USER" envDefault:"root"`
		Pass     string `env:"DB_PASS" envDefault:"password"`
		Database string `env:"DB_NAME" envDefault:"goapiboilerplate"`

		ConnMaxLifetime time.Duration `env:"MYSQL_CONN_MAX_LIFETIME" envDefault:"5m"` //  sets the maximum amount of time a connection may be reused
		MaxIdleConns    int           `env:"MYSQL_MAX_IDLE_CONNS" envDefault:"0"`     // sets the maximum number of connections in the idle
		MaxOpenConns    int           `env:"MYSQL_MAX_OPEN_CONNS" envDefault:"5"`     // sets the maximum number of connections in the idle
	}
	User struct {
		Host         string `env:"USER_HOST"          envDefault:"0.0.0.0"`
		ClientID     string `env:"USER_CLIENT_ID"     envDefault:"clientId"`
		ClientSecret string `env:"USER_CLIENT_SECRET" envDefault:"clientSecret"`
	}
	PubSub struct {
		Host string `env:"PUBSUB_HOST" envDefault:"0.0.0.0"`
		Port int    `env:"PUBSUB_HTTP" envDefault:"3001"`
	}
	CommandBus struct {
		QueueSize int `env:"COMMAND_BUS_BUFFER" envDefault:"0"`
	}
}

func init() {
	Env = &environment{}
	env.Parse(Env)

	if Env.CommandBus.QueueSize == 0 {
		Env.CommandBus.QueueSize = runtime.NumCPU()
	}

	log.Printf("config init : Env :\n%v\n", Env)
}
