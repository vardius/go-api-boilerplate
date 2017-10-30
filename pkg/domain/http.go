package domain

import (
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/gorouter"
)

// ErrEmptyRequestBody is when an request has empty body.
var ErrEmptyRequestBody = errors.New("Empty request body")

// ErrInvalidURLParams is when an request has invalid or missing parameters.
var ErrInvalidURLParams = errors.New("Invalid request URL params")

type dispatcher struct {
	commandBus CommandBus
}

func (d *dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var e error

	if r.Body == nil {
		r.WithContext(response.WithPayload(r, response.HTTPError{http.StatusBadRequest, ErrEmptyRequestBody, "Empty request body"}))
		return
	}

	params, ok := gorouter.FromContext(r.Context())
	if !ok {
		r.WithContext(response.WithPayload(r, response.HTTPError{http.StatusBadRequest, ErrInvalidURLParams, "Invalid URL params"}))
		return
	}

	defer r.Body.Close()
	body, e := ioutil.ReadAll(r.Body)
	if e != nil {
		r.WithContext(response.WithPayload(r, response.HTTPError{http.StatusBadRequest, e, "Invalid request body"}))
		return
	}

	out := make(chan error)
	defer close(out)

	go func() {
		d.commandBus.Publish(
			r.Context(),
			params.Value("domain")+params.Value("command"),
			body,
			out,
		)
	}()

	if e = <-out; e != nil {
		r.WithContext(response.WithPayload(r, response.HTTPError{http.StatusBadRequest, e, "Invalid request"}))
		return
	}

	w.WriteHeader(http.StatusCreated)

	return
}

// NewDispatcher creates handler for command bus
func NewDispatcher(cb CommandBus) http.Handler {
	return &dispatcher{cb}
}
