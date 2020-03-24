package metadata

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/vardius/gorouter/v4"

	md "github.com/vardius/go-api-boilerplate/pkg/metadata"
)

// WithMetadata adds Metadata to requests context
func WithMetadata() gorouter.MiddlewareFunc {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// Set the context with the required values to
			// process the request.
			m := md.Metadata{
				TraceID: uuid.New().String(),
				Now:     time.Now(),
			}

			ctx := md.ContextWithMetadata(r.Context(), &m)

			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(fn)
	}

	return m
}
