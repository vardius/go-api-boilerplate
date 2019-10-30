/*
Package recovery allows to recover from panic
*/
package recovery

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime/debug"

	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/golog"
)

var logger golog.Logger

// WithRecover recovers from panic
func WithRecover(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if logger != nil {
					logger.Critical(r.Context(), "[HTTP] Recovered in %v\n%s", rec, debug.Stack())
				}

				writeError(r.Context(), w)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// WithLogger registers logger
func WithLogger(l golog.Logger) {
	logger = l
}

func writeError(ctx context.Context, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(response.NewErrorFromHTTPStatus(http.StatusInternalServerError))
}
