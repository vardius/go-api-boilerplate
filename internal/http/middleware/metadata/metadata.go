package metadata

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	gorouter "github.com/vardius/gorouter/v4"
)

// ctxMetadataKey represents the type of value for the context key.
type ctxMetadataKey int

// KeyMetadataValues is how request values or stored/retrieved.
const KeyMetadataValues ctxMetadataKey = 1

// Metadata represent state for each request.
type Metadata struct {
	TraceID    string
	Now        time.Time
	StatusCode int
}

// WithMetadata adds Metadata to requests context
func WithMetadata() gorouter.MiddlewareFunc {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// Set the context with the required values to
			// process the request.
			m := Metadata{
				TraceID: uuid.New().String(),
				Now:     time.Now(),
			}

			r.WithContext(context.WithValue(r.Context(), KeyMetadataValues, &m))

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}

	return m
}
