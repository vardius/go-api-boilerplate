/*
Package log provides Logger
*/
package log

import "github.com/vardius/golog"

// Logger allow to create logger based on env setting
type Logger struct {
	golog.Logger
}

func getLogLevelByEnv(env string) string {
	logLevel := "info"
	if env == "development" {
		logLevel = "debug"
	}

	return logLevel
}

// New creates new logger based on environment
func New(env string) *Logger {
	var l golog.Logger
	if env == "development" {
		l = golog.New(getLogLevelByEnv(env))
	} else {
		l = golog.NewFileLogger(getLogLevelByEnv(env), "/tmp/prod.log")
	}

	return &Logger{l}
}
