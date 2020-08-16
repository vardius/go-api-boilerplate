package handlers

import (
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"

	"github.com/vardius/gorouter/v4/context"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// BuildCommandDispatchHandler wraps user gRPC client with http.Handler
func BuildCommandDispatchHandler(cb commandbus.CommandBus) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		if r.Body == nil {
			response.MustJSONError(r.Context(), w, ErrEmptyRequestBody)
			return
		}

		params, ok := context.Parameters(r.Context())
		if !ok {
			response.MustJSONError(r.Context(), w, ErrInvalidURLParams)
			return
		}

		defer r.Body.Close()
		body, e := ioutil.ReadAll(r.Body)
		if e != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(e))
			return
		}

		c, e := user.NewCommandFromPayload(params.Value("command"), body)
		if e != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(e))
			return
		}

		out := make(chan error, 1)
		defer close(out)

		go func() {
			out <- cb.Publish(r.Context(), c)
		}()

		ctxDoneCh := r.Context().Done()
		select {
		case <-ctxDoneCh:
			response.MustJSONError(r.Context(), w, errors.Wrap(fmt.Errorf("%w: %s", application.ErrTimeout, r.Context().Err())))
			return
		case e = <-out:
			if e != nil {
				response.MustJSONError(r.Context(), w, errors.Wrap(e))
				return
			}
		}

		w.WriteHeader(http.StatusCreated)
		if err := response.JSON(r.Context(), w, nil); err != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(err))
		}
	}

	return http.HandlerFunc(fn)
}

// BuildMeHandler wraps user gRPC client with http.Handler
func BuildMeHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		i, _ := identity.FromContext(r.Context())

		u, e := repository.Get(r.Context(), i.ID.String())
		if e != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(fmt.Errorf("%w: %s", application.ErrNotFound, e)))
			return
		}

		w.WriteHeader(http.StatusOK)
		if err := response.JSON(r.Context(), w, u); err != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(err))
		}
	}

	return http.HandlerFunc(fn)
}

// BuildGetUserHandler wraps user gRPC client with http.Handler
func BuildGetUserHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		params, ok := context.Parameters(r.Context())
		if !ok {
			response.MustJSONError(r.Context(), w, ErrInvalidURLParams)
			return
		}

		u, e := repository.Get(r.Context(), params.Value("id"))
		if e != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(fmt.Errorf("%w: %s", application.ErrNotFound, e)))
			return
		}

		w.WriteHeader(http.StatusOK)
		if err := response.JSON(r.Context(), w, u); err != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(err))
		}
	}

	return http.HandlerFunc(fn)
}

// BuildListUserHandler wraps user gRPC client with http.Handler
func BuildListUserHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		pageInt, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
		limitInt, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
		page := int32(math.Max(float64(pageInt), 1))
		limit := int32(math.Max(float64(limitInt), 20))

		totalUsers, e := repository.Count(r.Context())
		if e != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(e))
			return
		}

		offset := (page * limit) - limit

		paginatedList := struct {
			Users []persistence.User `json:"users"`
			Page  int32              `json:"page"`
			Limit int32              `json:"limit"`
			Total int32              `json:"total"`
		}{
			Page:  page,
			Limit: limit,
			Total: totalUsers,
		}

		if totalUsers < 1 || offset > (totalUsers-1) {
			w.WriteHeader(http.StatusOK)
			if err := response.JSON(r.Context(), w, paginatedList); err != nil {
				response.MustJSONError(r.Context(), w, errors.Wrap(err))
			}
			return
		}

		paginatedList.Users, e = repository.FindAll(r.Context(), limit, offset)
		if e != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(e))
			return
		}

		w.WriteHeader(http.StatusOK)
		if err := response.JSON(r.Context(), w, paginatedList); err != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(err))
		}
	}

	return http.HandlerFunc(fn)
}
