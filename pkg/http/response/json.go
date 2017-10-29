package response

import (
	"encoding/json"
	"net/http"
)

// JSON wraps handler and parse payload to json response
func JSON(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		next.ServeHTTP(w, r)

		if response, ok := fromContext(r.Context()); ok {
			encoder := json.NewEncoder(w)
			encoder.SetEscapeHTML(true)
			encoder.SetIndent("", "")

			err := encoder.Encode(response)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(HTTPError{
					Code:    http.StatusInternalServerError,
					Error:   err,
					Message: http.StatusText(http.StatusInternalServerError),
				})
				return
			}

			switch t := response.(type) {
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
