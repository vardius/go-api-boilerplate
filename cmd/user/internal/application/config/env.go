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
	App struct {
		Environment     string        `env:"APP_ENV"              envDefault:"development"`
		ShutdownTimeout time.Duration `env:"APP_SHUTDOWN_TIMEOUT" envDefault:"5s"`
		Secret          string        `env:"APP_SECRET"           envDefault:"secret"`

		ClientID     string `env:"USER_CLIENT_ID"     envDefault:"clientId"`
		ClientSecret string `env:"USER_CLIENT_SECRET" envDefault:"clientSecret"`
	}
	Facebook struct {
		ClientID     string `env:"USER_CLIENT_ID"     envDefault:"clientFacebookId"`
		ClientSecret string `env:"USER_CLIENT_SECRET" envDefault:"clientFacebookSecret"`
	}
	Google struct {
		ClientID     string `env:"USER_CLIENT_ID"     envDefault:"clientGoogleId"`
		ClientSecret string `env:"USER_CLIENT_SECRET" envDefault:"clientGoogleSecret"`
	}
	Debug struct {
		Host string `env:"DEBUG_HOST" envDefault:"0.0.0.0"`
		Port int    `env:"DEBUG_PORT" envDefault:"4000"`
	}
	HTTP struct {
		Host    string   `env:"HOST"         envDefault:"0.0.0.0"`
		Port    int      `env:"HTTP_PORT"    envDefault:"3000"`
		Origins []string `env:"HTTP_ORIGINS" envSeparator:"|"` // Origins should follow format: scheme "://" host [ ":" port ]t int    `env:"DEBUG_PORT_HTTP" envDefault:"4000"`

		ReadTimeout  time.Duration `env:"HTTP_SERVER_READ_TIMEOUT"     envDefault:"5s"`
		WriteTimeout time.Duration `env:"HTTP_SERVER_WRITE_TIMEOUT"    envDefault:"10s"`
		IdleTimeout  time.Duration `env:"HTTP_SERVER_SHUTDOWN_TIMEOUT" envDefault:"120s"`
	}
	GRPC struct {
		Host string `env:"HOST"      envDefault:"0.0.0.0"`
		Port int    `env:"GRPC_PORT" envDefault:"3001"`

		ServerMinTime time.Duration `env:"GRPC_SERVER_MIN_TIME" envDefault:"5m"` // if a client pings more than once every 5 minutes (default), terminate the connection
		ServerTime    time.Duration `env:"GRPC_SERVER_TIME" envDefault:"2h"`     // ping the client if it is idle for 2 hours (default) to ensure the connection is still active
		ServerTimeout time.Duration `env:"GRPC_SERVER_TIMEOUT" envDefault:"20s"` // wait 20 second (default) for the ping ack before assuming the connection is dead
		ConnTime      time.Duration `env:"GRPC_CONN_TIME" envDefault:"10s"`      // send pings every 10 seconds if there is no activity
		ConnTimeout   time.Duration `env:"GRPC_CONN_TIMEOUT" envDefault:"20s"`   // wait 20 second for ping ack before considering the connection dead
	}
	MYSQL struct {
		Host     string `env:"MYSQL_HOST" envDefault:"0.0.0.0"`
		Port     int    `env:"MYSQL_PORT" envDefault:"3306"`
		User     string `env:"MYSQL_USER" envDefault:"root"`
		Pass     string `env:"MYSQL_PASS" envDefault:"password"`
		Database string `env:"MYSQL_NAME" envDefault:"goapiboilerplate"`

		ConnMaxLifetime time.Duration `env:"MYSQL_CONN_MAX_LIFETIME" envDefault:"5m"` //  sets the maximum amount of time a connection may be reused
		MaxIdleConns    int           `env:"MYSQL_MAX_IDLE_CONNS" envDefault:"0"`     // sets the maximum number of connections in the idle
		MaxOpenConns    int           `env:"MYSQL_MAX_OPEN_CONNS" envDefault:"5"`     // sets the maximum number of connections in the idle
	}
	User struct {
		Host         string `env:"USER_HOST"          envDefault:"0.0.0.0"`
		ClientID     string `env:"USER_CLIENT_ID"     envDefault:"clientId"`
		ClientSecret string `env:"USER_CLIENT_SECRET" envDefault:"clientSecret"`
	}
	Auth struct {
		Host string `env:"AUTH_HOST" envDefault:"0.0.0.0"` // Auth service host
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

	if err := env.Parse(&Env.App); err != nil {
		panic(err)
	}
	if err := env.Parse(&Env.Debug); err != nil {
		panic(err)
	}
	if err := env.Parse(&Env.HTTP); err != nil {
		panic(err)
	}
	if err := env.Parse(&Env.GRPC); err != nil {
		panic(err)
	}
	if err := env.Parse(&Env.MYSQL); err != nil {
		panic(err)
	}
	if err := env.Parse(&Env.Auth); err != nil {
		panic(err)
	}
	if err := env.Parse(&Env.PubSub); err != nil {
		panic(err)
	}
	if err := env.Parse(&Env.CommandBus); err != nil {
		panic(err)
	}

	if Env.CommandBus.QueueSize == 0 {
		Env.CommandBus.QueueSize = runtime.NumCPU()
	}

	log.Printf("Env:\n%v\n", Env)
}
