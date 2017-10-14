package middleware

import (
	"context"
	"encoding/json"
	"net/http"
)

type responseKey struct{}

// NewContextWithResponse adds response do request context allowing later on Body middleware take care of it
func NewContextWithResponse(req *http.Request, i interface{}) context.Context {
	return context.WithValue(req.Context(), responseKey{}, i)
}

func responseFromContext(ctx context.Context) (interface{}, bool) {
	i := ctx.Value(responseKey{})
	return i, i != nil
}

// JSONBody parses response to json body
func JSONBody(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)

		if response, ok := responseFromContext(r.Context()); ok {
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
