/*
Package response provides helpers and utils for working with HTTP response
*/
package response

import (
	"context"
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/http/middleware/metadata"
)

// WriteHeader sends an HTTP response header with the provided status code,
// and sets status code on context's metadata
func WriteHeader(ctx context.Context, w http.ResponseWriter, statusCode int) {
	w.WriteHeader(statusCode)

	if mtd, ok := ctx.Value(metadata.KeyMetadataValues).(*metadata.Metadata); ok {
		mtd.StatusCode = statusCode
	}
}

// Flush checks if it is stream response and sends any buffered data to the client.
func Flush(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
