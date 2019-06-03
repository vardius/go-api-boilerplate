/*
Package recovery allows to recover from panic
*/
package recovery

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/golog"
)

func writeError(ctx context.Context, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(response.NewErrorFromHTTPStatus(http.StatusInternalServerError))
}

// WithRecover recovers from panic
func WithRecover(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				writeError(r.Context(), w)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// A Recover recovers http handler from panic
type Recover interface {
	RecoverHandler(next http.Handler) http.Handler
}

type loggableRecover struct {
	log golog.Logger
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

// WithLogger returns recover that logs info and recovers from panic
func WithLogger(log golog.Logger) Recover {
	return &loggableRecover{log}
}
