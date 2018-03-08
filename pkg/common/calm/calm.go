/*
Package calm allows to recover from panic
*/
package calm

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/common/http/response"
	"github.com/vardius/golog"
)

// A Recover recovers http handler from panic
type Recover interface {
	RecoverHandler(next http.Handler) http.Handler
}

func writeError(ctx context.Context, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(response.HTTPError{
		Code:    http.StatusInternalServerError,
		Message: http.StatusText(http.StatusInternalServerError),
	})
}

type defaultRecover int

func (r *defaultRecover) RecoverHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				writeError(req.Context(), w)
			}
		}()

		next.ServeHTTP(w, req)
	}

	return http.HandlerFunc(fn)
}

// New creates new panic recover middleware
func New() Recover {
	return new(defaultRecover)
}

type loggableRecover struct {
	recover Recover
	log     golog.Logger
}

func (r *loggableRecover) RecoverHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				r.log.Critical(req.Context(), "Recovered in f %v", rec)
				writeError(req.Context(), w)
			}
		}()

		next.ServeHTTP(w, req)
	}

	return http.HandlerFunc(fn)
}

// WithLogger returns copy of parent recover will also log info
func WithLogger(parent Recover, log golog.Logger) Recover {
	return &loggableRecover{parent, log}
}
