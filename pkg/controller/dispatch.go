package controller

import (
	"app/pkg/domain"
	"app/pkg/middleware"
	"io/ioutil"
	"net/http"

	"github.com/vardius/gorouter"
)

func CommandDispatch(commandBus domain.CommandBus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		if r.Body == nil {
			r.WithContext(middleware.NewContextWithResponse(r, &middleware.HTTPError{http.StatusBadRequest, ErrEmptyRequestBody, "Empty request body"}))
			return
		}

		params, ok := gorouter.FromContext(r.Context())
		if !ok {
			r.WithContext(middleware.NewContextWithResponse(r, &middleware.HTTPError{http.StatusBadRequest, ErrInvalidUrlParams, "Invalid URL params"}))
			return
		}

		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			r.WithContext(middleware.NewContextWithResponse(r, &middleware.HTTPError{http.StatusBadRequest, err, "Invalid request body"}))
			return
		}

		out := make(chan error)
		defer close(out)

		go func() {
			commandBus.Publish(
				params.Value("domain")+params.Value("command"),
				r.Context(),
				body,
				out,
			)
		}()

		if err = <-out; err != nil {
			r.WithContext(middleware.NewContextWithResponse(r, &middleware.HTTPError{http.StatusBadRequest, err, "Invalid request"}))
			return
		}

		w.WriteHeader(http.StatusCreated)

		return
	}
}
