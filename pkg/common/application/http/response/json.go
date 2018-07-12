package response

import (
	"encoding/json"
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/common/application/errors"
)

// AsJSON wraps handler and parse payload to json response
func AsJSON(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		ctx := contextWithResponse(r.Context())

		next.ServeHTTP(w, r.WithContext(ctx))

		if response, ok := fromContext(ctx); ok {
			if response.payload != nil {
				switch t := response.payload.(type) {
				case error:
					w.WriteHeader(errors.HTTPStatusCode(t))
				}

				encoder := json.NewEncoder(w)
				encoder.SetEscapeHTML(true)
				encoder.SetIndent("", "")

				err := encoder.Encode(response.payload)

				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					encoder.Encode(NewErrorFromHTTPStatus(http.StatusInternalServerError))

					return
				}
			}

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			} else {
				// Write nil in case of setting http.StatusOK header if header not set
				w.Write(nil)
			}
		}
	}

	return http.HandlerFunc(fn)
}
