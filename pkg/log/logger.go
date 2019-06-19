/*
Package log provides Logger
*/
package log

import (
	"net/http"
	"time"

	"github.com/vardius/golog"
)

// Logger allow to create logger based on env setting
type Logger struct {
	golog.Logger
}

// LogRequest wraps http.Handler with a logger middleware
func (l *Logger) LogRequest(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		l.Info(r.Context(), "[Request|Start]: %s %s\n", r.Method, r.URL.String())
		start := time.Now()
		next.ServeHTTP(w, r)
		l.Info(r.Context(), "[Request|End]: %s %s %s\n", r.Method, r.URL.String(), time.Since(start).String())
	}

	return http.HandlerFunc(fn)
}

// New creates new logger based on environment
func New(env string) *Logger {
	var l golog.Logger
	if env == "development" {
		l = golog.New(golog.Debug)
	} else {
		l = golog.NewFileLogger(golog.Info, "/tmp/prod.log")
	}

	return &Logger{l}
}
