package json

import (
	"app/pkg/err"
	"context"
	baseJson "encoding/json"
	"net/http"
)

type responseKey struct{}

// WithResponse adds response to request context allowing to middleware take care of it
func WithResponse(req *http.Request, i interface{}) context.Context {
	return context.WithValue(req.Context(), responseKey{}, i)
}

func fromContext(ctx context.Context) (interface{}, bool) {
	i := ctx.Value(responseKey{})
	return i, i != nil
}

// Parse response to json body
func Parse(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		next.ServeHTTP(w, r)

		if response, ok := fromContext(r.Context()); ok {
			encoder := baseJson.NewEncoder(w)
			encoder.SetEscapeHTML(true)
			encoder.SetIndent("", "")

			e := encoder.Encode(response)
			if e != nil {
				w.WriteHeader(http.StatusInternalServerError)
				baseJson.NewEncoder(w).Encode(err.HTTPError{
					Code:    http.StatusInternalServerError,
					Error:   e,
					Message: http.StatusText(http.StatusInternalServerError),
				})
				return
			}

			switch t := response.(type) {
			case err.HTTPError:
				w.WriteHeader(t.Code)
			case *err.HTTPError:
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
