/*
Package log provides Logger
*/
package log

import (
	"log"

	"github.com/vardius/golog"
)

// Logger allow to create logger based on env setting
type Logger struct {
	golog.Logger
}

// New creates new logger based on environment
func New(env string) *Logger {
	l := golog.New()
	if env != "development" {
		l.SetVerbosity(golog.DefaultVerbosity &^ golog.Debug)
	}

	l.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC)

	return &Logger{l}
}
