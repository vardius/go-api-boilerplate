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

	// If there is something to marshal otherwise set status code and do not marshal
	if payload != nil && statusCode != http.StatusNoContent {
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(true)
		encoder.SetIndent("", "")

		err := encoder.Encode(payload)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			RespondJSONError(ctx, w, errors.New(errors.INTERNAL, "Could not parse response to JSON."))

			return
		}
	}

	w.WriteHeader(statusCode)

	if metadata, ok := ctx.Value(metadata.KeyMetadataValues).(*metadata.Metadata); ok {
		metadata.StatusCode = statusCode
	}

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
