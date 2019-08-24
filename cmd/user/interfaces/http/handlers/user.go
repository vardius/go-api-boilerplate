package handlers

import (
	"io/ioutil"
	"math"
	"net/http"
	"strconv"

	"github.com/vardius/go-api-boilerplate/cmd/user/domain/user"
	"github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
	"github.com/vardius/gorouter/v4"
)

// BuildCommandDispatchHandler wraps user gRPC client with http.Handler
func BuildCommandDispatchHandler(cb commandbus.CommandBus) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		if r.Body == nil {
			response.WithError(r.Context(), ErrEmptyRequestBody)
			return
		}

		params, ok := gorouter.FromContext(r.Context())
		if !ok {
			response.WithError(r.Context(), ErrInvalidURLParams)
			return
		}

		defer r.Body.Close()
		body, e := ioutil.ReadAll(r.Body)
		if e != nil {
			response.WithError(r.Context(), errors.Wrap(e, errors.INTERNAL, "Invalid request body"))
			return
		}

		c, e := user.NewCommandFromPayload(params.Value("command"), body)
		if e != nil {
			response.WithError(r.Context(), errors.Wrap(e, errors.INTERNAL, "Invalid command payload"))
			return
		}

		out := make(chan error)
		defer close(out)

		go func() {
			cb.Publish(r.Context(), c, out)
		}()

		select {
		case <-r.Context().Done():
			response.WithError(r.Context(), errors.Wrap(r.Context().Err(), errors.TIMEOUT, "Request timeout"))
			return
		case e = <-out:
			if e != nil {
				response.WithError(r.Context(), errors.Wrap(e, errors.INTERNAL, "Command handler error"))
				return
			}
		}

		w.WriteHeader(http.StatusCreated)

		return
	}

	return http.HandlerFunc(fn)
}

// BuildMeHandler wraps user gRPC client with http.Handler
func BuildMeHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		if r.Body == nil {
			response.WithError(r.Context(), ErrEmptyRequestBody)
			return
		}

		i, _ := identity.FromContext(r.Context())

		user, e := repository.Get(r.Context(), i.ID.String())
		if e != nil {
			response.WithError(r.Context(), errors.Wrap(e, errors.NOTFOUND, "User not found"))
			return
		}

		response.WithPayload(r.Context(), user)
		return
	}

	return http.HandlerFunc(fn)
}

// BuildGetUserHandler wraps user gRPC client with http.Handler
func BuildGetUserHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		if r.Body == nil {
			response.WithError(r.Context(), ErrEmptyRequestBody)
			return
		}

		params, ok := gorouter.FromContext(r.Context())
		if !ok {
			response.WithError(r.Context(), ErrInvalidURLParams)
			return
		}

		user, e := repository.Get(r.Context(), params.Value("id"))
		if e != nil {
			response.WithError(r.Context(), errors.Wrap(e, errors.NOTFOUND, "User not found"))
			return
		}

		response.WithPayload(r.Context(), user)
		return
	}

	return http.HandlerFunc(fn)
}

// BuildListUserHandler wraps user gRPC client with http.Handler
func BuildListUserHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		if r.Body == nil {
			response.WithError(r.Context(), ErrEmptyRequestBody)
			return
		}

		pageInt, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
		limitInt, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
		page := int32(math.Max(float64(pageInt), 1))
		limit := int32(math.Max(float64(limitInt), 20))

		totalUsers, e := repository.Count(r.Context())
		if e != nil {
			response.WithError(r.Context(), errors.Wrap(e, errors.INTERNAL, "Invalid request"))
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
			response.WithPayload(r.Context(), paginatedList)
			return
		}

		paginatedList.Users, e = repository.FindAll(r.Context(), limit, offset)
		if e != nil {
			response.WithError(r.Context(), errors.Wrap(e, errors.INTERNAL, "Invalid request"))
			return
		}

		response.WithPayload(r.Context(), paginatedList)
		return
	}

	return http.HandlerFunc(fn)
}
