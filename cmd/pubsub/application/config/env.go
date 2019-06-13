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
	Environment string `env:"ENV" envDefault:"development"`

	Host string `env:"HOST" envDefault:"0.0.0.0"`
	Port int    `env:"PORT" envDefault:"3001"`

	QueueSize int `env:"QUEUE_BUFFER_SIZE" envDefault:"0"`
}

func init() {
	once.Do(func() {
		Env = &environment{}
		env.Parse(Env)

		if Env.QueueSize == 0 {
			Env.QueueSize = runtime.NumCPU()
		}
	})
}
