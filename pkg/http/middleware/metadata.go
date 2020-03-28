package middleware

import (
	"net/http"

	"github.com/vardius/gorouter/v4"

	md "github.com/vardius/go-api-boilerplate/pkg/metadata"
)

// WithMetadata adds Metadata to requests context
func WithMetadata() gorouter.MiddlewareFunc {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// Set the context with the required values to
			// process the request.
			mtd := md.New()

			ctx := md.ContextWithMetadata(r.Context(), mtd)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}

	return m
}
