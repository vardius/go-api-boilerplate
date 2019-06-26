/*
Package log provides Logger
*/
package log

import (
	"log"
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
	l := golog.New()
	if env != "development" {
		l.SetVerbosity(golog.DefaultVerbosity &^ golog.Debug)
	}

	l.SetFlags(log.Llongfile | log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC)

	return &Logger{l}
}
