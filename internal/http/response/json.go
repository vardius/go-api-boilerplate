/*
Package response provides helpers and utils for working with HTTP response
*/
package response

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/vardius/go-api-boilerplate/internal/errors"
)

// WithPayloadAsJSON adds payload to context for response
func WithPayloadAsJSON(ctx context.Context, w http.ResponseWriter, payload interface{}, statusCode int) {

	// If there is something to marshal otherwise set status code and do not marshal
	if payload != nil && statusCode != http.StatusNoContent {
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(true)
		encoder.SetIndent("", "")

		err := encoder.Encode(payload)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			encoder.Encode(NewErrorFromHTTPStatus(http.StatusInternalServerError))

			return
		}
	}

	w.WriteHeader(statusCode)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	} else {
		// Write nil in case of setting http.StatusOK header if header not set
		w.Write(nil)
	}
}

// WithErrorAsJSON adds error to context for response
// uses WithPayloadAsJSON internally
func WithErrorAsJSON(ctx context.Context, w http.ResponseWriter, err error) {
	WithPayloadAsJSON(ctx, w, err, errors.HTTPStatusCode(err))
}
