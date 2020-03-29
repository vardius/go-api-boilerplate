package handlers

import (
	"io/ioutil"
	"math"
	"net/http"
	"strconv"

	"github.com/vardius/gorouter/v4/context"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
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
			w.WriteHeader(errors.HTTPStatusCode(ErrEmptyRequestBody))

			if err := response.JSON(r.Context(), w, ErrEmptyRequestBody); err != nil {
				panic(err)
			}
			return
		}

		params, ok := context.Parameters(r.Context())
		if !ok {
			w.WriteHeader(errors.HTTPStatusCode(ErrInvalidURLParams))

			if err := response.JSON(r.Context(), w, ErrInvalidURLParams); err != nil {
				panic(err)
			}
			return
		}

		defer r.Body.Close()
		body, e := ioutil.ReadAll(r.Body)
		if e != nil {
			appErr := errors.Wrap(e, errors.INTERNAL, "Invalid request body")
			w.WriteHeader(errors.HTTPStatusCode(appErr))

			if err := response.JSON(r.Context(), w, appErr); err != nil {
				panic(err)
			}
			return
		}

		c, e := user.NewCommandFromPayload(params.Value("command"), body)
		if e != nil {
			appErr := errors.Wrap(e, errors.INTERNAL, errors.ErrorMessage(e))
			w.WriteHeader(errors.HTTPStatusCode(appErr))

			if err := response.JSON(r.Context(), w, appErr); err != nil {
				panic(err)
			}
			return
		}

		out := make(chan error, 1)
		defer close(out)

		go func() {
			cb.Publish(r.Context(), c, out)
		}()

		select {
		case <-r.Context().Done():
			appErr := errors.Wrap(r.Context().Err(), errors.TIMEOUT, "Request timeout")
			w.WriteHeader(errors.HTTPStatusCode(appErr))

			if err := response.JSON(r.Context(), w, appErr); err != nil {
				panic(err)
			}
			return
		case e = <-out:
			if e != nil {
				appErr := errors.Wrap(e, errors.INTERNAL, "Command handler error")
				w.WriteHeader(errors.HTTPStatusCode(appErr))

				if err := response.JSON(r.Context(), w, appErr); err != nil {
					panic(err)
				}
				return
			}
		}

		w.WriteHeader(http.StatusCreated)

		if err := response.JSON(r.Context(), w, nil); err != nil {
			panic(err)
		}
	}

	return http.HandlerFunc(fn)
}

// BuildMeHandler wraps user gRPC client with http.Handler
func BuildMeHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		if r.Body == nil {
			w.WriteHeader(errors.HTTPStatusCode(ErrEmptyRequestBody))

			if err := response.JSON(r.Context(), w, ErrEmptyRequestBody); err != nil {
				panic(err)
			}
			return
		}

		i, _ := identity.FromContext(r.Context())

		u, e := repository.Get(r.Context(), i.ID.String())
		if e != nil {
			appErr := errors.Wrap(e, errors.NOTFOUND, "User not found")
			w.WriteHeader(errors.HTTPStatusCode(appErr))

			if err := response.JSON(r.Context(), w, appErr); err != nil {
				panic(err)
			}
			return
		}

		w.WriteHeader(http.StatusOK)

		if err := response.JSON(r.Context(), w, u); err != nil {
			panic(err)
		}
		return
	}

	return http.HandlerFunc(fn)
}

// BuildGetUserHandler wraps user gRPC client with http.Handler
func BuildGetUserHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		if r.Body == nil {
			appErr := ErrEmptyRequestBody
			w.WriteHeader(errors.HTTPStatusCode(appErr))

			if err := response.JSON(r.Context(), w, appErr); err != nil {
				panic(err)
			}
			return
		}

		params, ok := context.Parameters(r.Context())
		if !ok {
			appErr := ErrInvalidURLParams
			w.WriteHeader(errors.HTTPStatusCode(appErr))

			if err := response.JSON(r.Context(), w, appErr); err != nil {
				panic(err)
			}
			return
		}

		u, e := repository.Get(r.Context(), params.Value("id"))
		if e != nil {
			appErr := errors.Wrap(e, errors.NOTFOUND, "User not found")
			w.WriteHeader(errors.HTTPStatusCode(appErr))

			if err := response.JSON(r.Context(), w, appErr); err != nil {
				panic(err)
			}
			return
		}

		w.WriteHeader(http.StatusOK)

		if err := response.JSON(r.Context(), w, u); err != nil {
			panic(err)
		}
		return
	}

	return http.HandlerFunc(fn)
}

// BuildListUserHandler wraps user gRPC client with http.Handler
func BuildListUserHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		if r.Body == nil {
			w.WriteHeader(errors.HTTPStatusCode(ErrEmptyRequestBody))

			if err := response.JSON(r.Context(), w, ErrEmptyRequestBody); err != nil {
				panic(err)
			}
			return
		}

		pageInt, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
		limitInt, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
		page := int32(math.Max(float64(pageInt), 1))
		limit := int32(math.Max(float64(limitInt), 20))

		totalUsers, e := repository.Count(r.Context())
		if e != nil {
			appErr := errors.New(errors.INTERNAL, http.StatusText(http.StatusInternalServerError))
			w.WriteHeader(errors.HTTPStatusCode(appErr))

			if err := response.JSON(r.Context(), w, appErr); err != nil {
				panic(err)
			}
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
				panic(err)
			}
			return
		}

		paginatedList.Users, e = repository.FindAll(r.Context(), limit, offset)
		if e != nil {
			appErr := errors.New(errors.INTERNAL, http.StatusText(http.StatusInternalServerError))
			w.WriteHeader(errors.HTTPStatusCode(appErr))

			if err := response.JSON(r.Context(), w, appErr); err != nil {
				panic(err)
			}
			return
		}

		w.WriteHeader(http.StatusOK)

		if err := response.JSON(r.Context(), w, paginatedList); err != nil {
			panic(err)
		}
		return
	}

	return http.HandlerFunc(fn)
}
