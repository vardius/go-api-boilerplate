package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/vardius/gorouter/v4"

	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/request"
	json2 "github.com/vardius/go-api-boilerplate/pkg/http/response/json"
	md "github.com/vardius/go-api-boilerplate/pkg/metadata"
)

const InternalRequestMetadataKey = "m"

// responseWriter is a minimal wrapper for http.ResponseWriter that allows the
// written HTTP statusCode to be captured for metadata.
type responseWriter struct {
	http.ResponseWriter
	mtd         *md.Metadata
	wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter, mtd *md.Metadata) *responseWriter {
	return &responseWriter{ResponseWriter: w, mtd: mtd}
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	if rw.wroteHeader {
		return
	}

	rw.mtd.StatusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
	rw.wroteHeader = true

	return
}

// WithMetadata adds Metadata to requests context
func WithMetadata() gorouter.MiddlewareFunc {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			var mtd *md.Metadata
			if m := r.URL.Query().Get(InternalRequestMetadataKey); m != "" {
				data, err := base64.RawURLEncoding.DecodeString(m)
				if err != nil {
					json2.MustJSONError(r.Context(), w, apperrors.Wrap(err))
				}

				if err := json.Unmarshal(data, &mtd); err != nil {
					json2.MustJSONError(r.Context(), w, apperrors.Wrap(err))
				}
			} else {
				// set the context with the required values to
				// process the request.
				mtd = md.New()

				mtd.RemoteAddr = r.RemoteAddr
				mtd.UserAgent = r.UserAgent()
				mtd.Referer = r.Referer()
				mtd.StatusCode = http.StatusOK // default status code returned by net/http package, will be overridden by WriteHeader calls
				if ip, err := request.IpAddress(r); err == nil {
					mtd.IPAddress = ip
				}
			}

			ctx := md.ContextWithMetadata(r.Context(), mtd)

			next.ServeHTTP(wrapResponseWriter(w, mtd), r.WithContext(ctx))
		}

		return http.HandlerFunc(fn)
	}

	return m
}
