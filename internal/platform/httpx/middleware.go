package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
	"slices"
	"time"
	"uuid"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/obs"
)

// Middleware wraps a handler. It is the shape gorouter's middleware has.
type Middleware = func(http.Handler) http.Handler

// Chain composes mws into one middleware; the first is the outermost.
func Chain(mws ...Middleware) Middleware {
	return func(h http.Handler) http.Handler {
		for _, mw := range slices.Backward(mws) {
			h = mw(h)
		}
		return h
	}
}

// ServeWithContext calls next with r carrying ctx, then copies the pattern
// the router set on that request back to r, so that middleware further out
// can still read r.Pattern.
func ServeWithContext(ctx context.Context, next http.Handler, w http.ResponseWriter, r *http.Request) {
	routed := r.WithContext(ctx)
	defer func() { r.Pattern = routed.Pattern }()
	next.ServeHTTP(w, routed)
}

// Recover turns a panic in the handler into a 500 problem and logs it with
// its stack. When the response has already started, it aborts the
// connection instead, so the client sees a failure rather than a truncated
// success. http.ErrAbortHandler passes through: it is how a handler asks for
// exactly that.
func Recover(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &recorder{ResponseWriter: w}
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(v)
				}
				log.ErrorContext(r.Context(), "handler panicked",
					"panic", v,
					"request_id", rec.Header().Get(requestIDHeader),
					"stack", string(debug.Stack()),
				)
				if rec.wroteHeader {
					panic(http.ErrAbortHandler)
				}
				writeProblem(rec, r, http.StatusInternalServerError, "", nil)
			}()
			next.ServeHTTP(rec, r)
		})
	}
}

const requestIDHeader = "X-Request-Id"

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// RequestID gives every request an ID: the incoming X-Request-Id when it is
// 1 to 64 characters of [A-Za-z0-9._-], a new UUIDv7 otherwise. The ID is
// echoed in the response's X-Request-Id and carried by the context, where
// obs.RequestID reads it.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if !validRequestID.MatchString(id) {
			id = uuid.NewV7().String()
		}
		w.Header().Set(requestIDHeader, id)
		ServeWithContext(obs.WithRequestID(r.Context(), id), next, w, r)
	})
}

// Trace starts a server span for each request with the global OTel
// providers, continuing the caller's trace. Once the router has run, the
// span is named after the route, as in "GET /v1/clients/{id}".
func Trace(next http.Handler) http.Handler {
	return otelhttp.NewMiddleware("http")(next)
}

// Log writes one line per request: method, route pattern, status, bytes
// written and duration, plus the error the handler returned, if any. The
// line is at error level for a 5xx and at debug level for a request whose
// client left; obs.NewLogger adds the request and trace IDs.
func Log(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &recorder{ResponseWriter: w}
			completed := false
			defer func() {
				code := rec.status
				attrs := make([]slog.Attr, 0, 8)
				switch {
				case !completed:
					attrs = append(attrs, slog.Bool("panic", true))
					if !rec.wroteHeader {
						code = http.StatusInternalServerError
					}
				case !rec.wroteHeader:
					code = http.StatusOK
				}
				level := slog.LevelInfo
				switch {
				case code >= http.StatusInternalServerError:
					level = slog.LevelError
				case !rec.wroteHeader && errors.Is(rec.err, context.Canceled):
					level = slog.LevelDebug
				}
				attrs = append(attrs,
					slog.String("method", r.Method),
					slog.String("pattern", r.Pattern),
					slog.Int("status", code),
					slog.Int64("bytes", rec.bytes),
					slog.Duration("duration", time.Since(start)),
				)
				if rec.err != nil {
					attrs = append(attrs, slog.String("err", rec.err.Error()))
				}
				log.LogAttrs(r.Context(), level, "request", attrs...)
			}()
			next.ServeHTTP(rec, r)
			completed = true
		})
	}
}

// SecurityHeaders sets headers that suit a JSON API: no MIME sniffing, no
// referrer, no content or framing allowed, and no caching. A handler may
// override them, as a JWKS endpoint does with Cache-Control.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// BodyLimit caps request bodies at n bytes; reading past that is an
// *http.MaxBytesError, which WriteError answers with 413.
func BodyLimit(n int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body := r.Body
			r.Body = http.MaxBytesReader(w, body, n)
			// net/http inspects the body it created once the handler
			// returns, so give it back.
			defer func() { r.Body = body }()
			next.ServeHTTP(w, r)
		})
	}
}

// CrossOrigin rejects unsafe cross-origin browser requests with a 403
// problem, using http.CrossOriginProtection. It is what makes cookie
// authentication safe from cross-site request forgery.
func CrossOrigin(next http.Handler) http.Handler {
	c := http.NewCrossOriginProtection()
	c.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, r, apperr.New(apperr.ErrForbidden, "Cross-origin requests are not allowed."))
	}))
	return c.Handler(next)
}
