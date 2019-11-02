/*
Package response provides helpers and utils for working with HTTP response
*/
package response

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/vardius/go-api-boilerplate/internal/errors"
	"github.com/vardius/go-api-boilerplate/internal/http/middleware/metadata"
)

// RespondJSON returns data as json response
func RespondJSON(ctx context.Context, w http.ResponseWriter, payload interface{}, statusCode int) {

	// If there is nothing to marshal then set status code and return.
	if payload == nil || statusCode == http.StatusNoContent {
		w.WriteHeader(statusCode)
		if metadata, ok := ctx.Value(metadata.KeyMetadataValues).(*metadata.Metadata); ok {
			metadata.StatusCode = statusCode
		}
		return
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(true)
	encoder.SetIndent("", "")

	if err := encoder.Encode(payload); err != nil {
		panic(err)
	}

	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")

	if metadata, ok := ctx.Value(metadata.KeyMetadataValues).(*metadata.Metadata); ok {
		metadata.StatusCode = statusCode
	}

	// Check if it is stream response
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	} else {
		// Write nil in case of setting http.StatusOK header if header not set
		w.Write(nil)
	}
}

// RespondJSONError returns error response
// uses WithPayloadAsJSON internally
func RespondJSONError(ctx context.Context, w http.ResponseWriter, err error) {
	RespondJSON(ctx, w, err, errors.HTTPStatusCode(err))
}
