package response

import (
	"encoding/json"
	"net/http"
)

// AsJSON wraps handler and parse payload to json response
func AsJSON(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		r = r.WithContext(contextWithResponse(r.Context()))

		next.ServeHTTP(w, r)

		if response, ok := fromContext(r.Context()); ok {
			encoder := json.NewEncoder(w)
			encoder.SetEscapeHTML(true)
			encoder.SetIndent("", "")

			err := encoder.Encode(response.payload)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(HTTPError{
					Code:    http.StatusInternalServerError,
					Error:   err,
					Message: http.StatusText(http.StatusInternalServerError),
				})
				return
			}

			switch t := response.payload.(type) {
			case HTTPError:
				w.WriteHeader(t.Code)
			case *HTTPError:
				w.WriteHeader(t.Code)
			default:
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				} else {
					// Write nil in case of setting http.StatusOK header if header not set
					w.Write(nil)
				}
			}
		}
	}

	return http.HandlerFunc(fn)
}
