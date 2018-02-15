package response

import (
  "encoding/json"
  "net/http"
  "fmt"
)

// AsJSON wraps handler and parse payload to json response
func AsJSON(next http.Handler) http.Handler {
  fn := func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    ctx := contextWithResponse(r.Context())

    next.ServeHTTP(w, r.WithContext(ctx))

    if response, ok := fromContext(ctx); ok {
      switch t := response.payload.(type) {
      case HTTPError:
        w.WriteHeader(t.Code)
      case *HTTPError:
        w.WriteHeader(t.Code)
      }

      encoder := json.NewEncoder(w)
      encoder.SetEscapeHTML(true)
      encoder.SetIndent("", "")

      fmt.Println(response.payload)

      if response.payload != nil {
        err := encoder.Encode(response.payload)

        if err != nil {
          w.WriteHeader(http.StatusInternalServerError)
          encoder.Encode(HTTPError{
            Code:    http.StatusInternalServerError,
            Error:   err,
            Message: http.StatusText(http.StatusInternalServerError),
          })

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
