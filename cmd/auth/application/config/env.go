package config

import (
	"runtime"
	"time"

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

	MysqlHost string `env:"DB_HOST" envDefault:"0.0.0.0"`
	MysqlPort int    `env:"DB_PORT" envDefault:"3306"`
	MysqlUser string `env:"DB_USER" envDefault:"root"`
	MysqlPass string `env:"DB_PASS" envDefault:"password"`
	MysqlName string `env:"DB_NAME" envDefault:"goapiboilerplate"`

	UserHost         string `env:"USER_HOST"          envDefault:"0.0.0.0"`
	UserClientID     string `env:"USER_CLIENT_ID"     envDefault:"clientId"`
	UserClientSecret string `env:"USER_CLIENT_SECRET" envDefault:"clientSecret"`

	PubSubHost string `env:"PUBSUB_HOST" envDefault:"0.0.0.0"`

	CommandBusQueueSize  int           `env:"USER_COMMAND_BUS_BUFFER" envDefault:"0"`
	GrpcServerMinTime    time.Duration `env:"USER_GRPC_SERVER_MIN_TIME" envDefault:"5m"`    // if a client pings more than once every 5 minutes (default), terminate the connection
	GrpcServerTime       time.Duration `env:"USER_GRPC_SERVER_TIME" envDefault:"2h"`        // ping the client if it is idle for 2 hours (default) to ensure the connection is still active
	GrpcServerTimeout    time.Duration `env:"USER_GRPC_SERVER_TIMEOUT" envDefault:"20s"`    // wait 20 second (default) for the ping ack before assuming the connection is dead
	GrpcConnTime         time.Duration `env:"USER_GRPC_CONN_TIME" envDefault:"10s"`         // send pings every 10 seconds if there is no activity
	GrpcConnTimeout      time.Duration `env:"USER_GRPC_CONN_TIMEOUT" envDefault:"20s"`      // wait 20 second for ping ack before considering the connection dead
	MysqlConnMaxLifetime time.Duration `env:"USER_MYSQL_CONN_MAX_LIFETIME" envDefault:"5m"` //  sets the maximum amount of time a connection may be reused
	MysqlMaxIdleConns    int           `env:"USER_MYSQL_MAX_IDLE_CONNS" envDefault:"0"`     // sets the maximum number of connections in the idle
	MysqlMaxOpenConns    int           `env:"USER_MYSQL_MAX_OPEN_CONNS" envDefault:"5"`     // sets the maximum number of connections in the idle
}

func (e *environment) GetMysqlHost() string                   { return e.MysqlHost }
func (e *environment) GetMysqlPort() int                      { return e.MysqlPort }
func (e *environment) GetMysqlUser() string                   { return e.MysqlUser }
func (e *environment) GetMysqlPass() string                   { return e.MysqlPass }
func (e *environment) GetMysqlDatabase() string               { return e.MysqlName }
func (e *environment) GetMysqlConnMaxLifetime() time.Duration { return e.MysqlConnMaxLifetime }
func (e *environment) GetMysqlMaxIdleConns() int              { return e.MysqlMaxIdleConns }
func (e *environment) GetMysqlMaxOpenConns() int              { return e.MysqlMaxOpenConns }

func (e *environment) GetGrpcServerMinTime() time.Duration { return e.GrpcServerMinTime }
func (e *environment) GetGrpcServerTime() time.Duration    { return e.GrpcServerTime }
func (e *environment) GetGrpcServerTimeout() time.Duration { return e.GrpcServerTimeout }

func (e *environment) GetGrpcConnTime() time.Duration    { return e.GrpcConnTime }
func (e *environment) GetGrpcConnTimeout() time.Duration { return e.GrpcConnTimeout }

func init() {
	Env = &environment{}
	env.Parse(Env)

	if Env.CommandBusQueueSize == 0 {
		Env.CommandBusQueueSize = runtime.NumCPU()
	}
}
