/*
Package response provides helpers and utils for working with HTTP response
*/
package response

import (
	"context"
	"encoding/json"
	"net/http"
)

// JSON returns data as json response
func JSON(ctx context.Context, w http.ResponseWriter, payload interface{}) error {
	w.Header().Set("Content-Type", "application/json")

	// If there is nothing to marshal then set status code and return.
	if payload == nil {
		WriteHeader(ctx, w, http.StatusNoContent)

		_, err := w.Write([]byte("{}"))
		return err
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(true)
	encoder.SetIndent("", "")

	if err := encoder.Encode(payload); err != nil {
		return err
	}

	Flush(w)

	return nil
}
