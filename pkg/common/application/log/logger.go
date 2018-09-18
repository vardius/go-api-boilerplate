/*
Package log provides Logger
*/
package log

import (
	"net/http"
	"time"

	"github.com/vardius/golog"
	"github.com/vardius/gorouter"
)

// Logger allow to create logger based on env setting
type Logger struct {
	golog.Logger
}

// LogRequest wraps http.Handler with a logger middleware
func (l *Logger) LogRequest(serverName string) gorouter.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			l.Info(r.Context(), "[%s Request|Start]: %s\n", r.Method, r.URL.String())
			start := time.Now()
			next.ServeHTTP(w, r)
			l.Info(r.Context(), "[%s Request|End] %s %q\n", r.Method, r.URL.String(), time.Since(start))
		}

		return http.HandlerFunc(fn)
	}
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
