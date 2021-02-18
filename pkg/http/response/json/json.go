/*
Package response provides helpers and utils for working with HTTP response
*/
package json

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	httperrors "github.com/vardius/go-api-boilerplate/pkg/http/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
)

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// ServeHTTP calls f(w, r) and handles error
func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := f(w, r); err != nil {
		MustJSONError(r.Context(), w, err)
	}
}

// JSON returns data as json response
func JSON(ctx context.Context, w http.ResponseWriter, statusCode int, payload interface{}) error {
	w.Header().Set("Content-Type", "application/json")

	// If there is nothing to marshal then set status code and return.
	if payload == nil {
		_, err := w.Write([]byte("{}"))
		return err
	}

	if statusCode != http.StatusOK {
		w.WriteHeader(statusCode)
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(true)
	encoder.SetIndent("", "")

	if err := encoder.Encode(payload); err != nil {
		return err
	}

	response.Flush(w)

	return nil
}

// MustJSON returns data as json response
// will panic if unable to marshal payload into JSON object
// uses JSON internally
func MustJSON(ctx context.Context, w http.ResponseWriter, statusCode int, payload interface{}) {
	if err := JSON(ctx, w, statusCode, payload); err != nil {
		panic(err)
	}
}

func NotFound() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		httpError := &httperrors.HttpError{
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("Route %s %s", r.URL.Path, http.StatusText(http.StatusNotFound)),
		}

		return JSON(r.Context(), w, httpError.Code, httpError)
	}

	return HandlerFunc(fn)
}

func NotAllowed() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		httpError := &httperrors.HttpError{
			Code:    http.StatusMethodNotAllowed,
			Message: http.StatusText(http.StatusMethodNotAllowed),
		}

		return JSON(r.Context(), w, httpError.Code, httpError)
	}

	return HandlerFunc(fn)
}
