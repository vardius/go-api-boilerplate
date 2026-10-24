package httpx

import (
	"cmp"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/obs"
)

// HandlerFunc is an HTTP handler that returns its failure instead of
// writing it.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// Handle adapts h to net/http, writing the error h returns with WriteError.
func Handle(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			WriteError(w, r, err)
		}
	}
}

// status is the one table from errors to status codes. 0 means the client
// has gone and nothing is written.
func status(err error) int {
	switch {
	case isMaxBytes(err):
		return http.StatusRequestEntityTooLarge
	case isTooManyRequests(err):
		return http.StatusTooManyRequests
	case errors.Is(err, apperr.ErrInvalid):
		return http.StatusBadRequest
	case errors.Is(err, apperr.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, apperr.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, apperr.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, apperr.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusServiceUnavailable
	case errors.Is(err, context.Canceled):
		return 0
	default:
		return http.StatusInternalServerError
	}
}

func isMaxBytes(err error) bool {
	_, ok := errors.AsType[*http.MaxBytesError](err)
	return ok
}

func isTooManyRequests(err error) bool {
	_, ok := errors.AsType[*TooManyRequestsError](err)
	return ok
}

// WriteError writes err as an RFC 9457 problem, with the status from its
// apperr kind. Only text set with apperr.New, or a ValidationError's fields,
// reaches the client; a 5xx says nothing beyond its status.
//
// When Log wraps the handler, err goes in the request's log line, at error
// level for a 5xx; without it, a 5xx is logged here. A response already
// started can't become a problem, so then err is only logged. A
// context.Canceled writes nothing: the client has left.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	code := status(err)
	rec := findRecorder(w)
	if rec != nil {
		rec.err = err
	} else {
		switch {
		case code == 0:
			slog.DebugContext(r.Context(), "request canceled", "err", err)
		case code >= http.StatusInternalServerError:
			slog.ErrorContext(r.Context(), "request failed", "err", err)
		}
	}
	if code == 0 || rec != nil && rec.wroteHeader {
		return
	}

	var detail string
	var fields map[string]string
	if code < http.StatusInternalServerError {
		detail = apperr.Message(err)
		if v, ok := errors.AsType[*apperr.ValidationError](err); ok {
			fields = v.Fields
		}
	}
	if mbe, ok := errors.AsType[*http.MaxBytesError](err); ok && detail == "" {
		detail = fmt.Sprintf("The request body is larger than %d bytes.", mbe.Limit)
	}
	if tmr, ok := errors.AsType[*TooManyRequestsError](err); ok {
		secs := tmr.retryAfterSeconds()
		w.Header().Set("Retry-After", strconv.Itoa(secs))
		if detail == "" {
			detail = fmt.Sprintf("Too many requests; try again in %s.", time.Duration(secs)*time.Second)
		}
	}
	if code == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", "Bearer")
	}
	writeProblem(w, r, code, detail, fields)
}

// RecordError puts err in the request's log line, as WriteError does, for
// a handler that answers the failure in a format of its own, such as an
// OAuth 2 error response. It writes nothing; without Log it does nothing.
func RecordError(w http.ResponseWriter, err error) {
	if rec := findRecorder(w); rec != nil {
		rec.err = err
	}
}

type problem struct {
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Status    int               `json:"status"`
	Detail    string            `json:"detail,omitzero"`
	Instance  string            `json:"instance,omitzero"`
	RequestID string            `json:"request_id,omitzero"`
	Errors    map[string]string `json:"errors,omitzero"`
}

func writeProblem(w http.ResponseWriter, r *http.Request, code int, detail string, fields map[string]string) {
	// A problem is strings and an int, so encoding it can't fail.
	body, _ := json.Marshal(problem{
		Type:     "about:blank",
		Title:    http.StatusText(code),
		Status:   code,
		Detail:   detail,
		Instance: r.URL.Path,
		// Recover runs outside RequestID, so its context has no ID yet;
		// the response header RequestID set does.
		RequestID: cmp.Or(obs.RequestID(r.Context()), w.Header().Get(requestIDHeader)),
		Errors:    fields,
	})
	h := w.Header()
	h.Set("Content-Type", "application/problem+json")
	h.Del("Content-Length")
	w.WriteHeader(code)
	_, _ = w.Write(append(body, '\n'))
}

// WriteJSON writes v as a JSON response with status. v is encoded before
// anything is written, so an encoding failure, which WriteJSON returns,
// leaves the response untouched for WriteError. A failed write means the
// client has gone, so it isn't reported.
func WriteJSON(w http.ResponseWriter, status int, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encode response: %w", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
	return nil
}
