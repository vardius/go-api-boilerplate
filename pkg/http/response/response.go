/*
Package response provides helpers and utils for working with HTTP response
*/
package response

import (
	"net/http"
)

// Flush checks if it is stream response and sends any buffered data to the client.
func Flush(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
