package handlers

import (
	"io/ioutil"
	"math"
	"net/http"
	"strconv"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/internal/commandbus"
	"github.com/vardius/go-api-boilerplate/internal/errors"
	"github.com/vardius/go-api-boilerplate/internal/http/response"
	"github.com/vardius/go-api-boilerplate/internal/identity"
	"github.com/vardius/gorouter/v4/context"
)

// BuildCommandDispatchHandler wraps user gRPC client with http.Handler
func BuildCommandDispatchHandler(cb commandbus.CommandBus) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		if r.Body == nil {
			response.RespondJSONError(r.Context(), w, ErrEmptyRequestBody)
			return
		}

		params, ok := context.Parameters(r.Context())
		if !ok {
			response.RespondJSONError(r.Context(), w, ErrInvalidURLParams)
			return
		}

		defer r.Body.Close()
		body, e := ioutil.ReadAll(r.Body)
		if e != nil {
			response.RespondJSONError(r.Context(), w, errors.Wrap(e, errors.INTERNAL, "Invalid request body"))
			return
		}

		c, e := user.NewCommandFromPayload(params.Value("command"), body)
		if e != nil {
			response.RespondJSONError(r.Context(), w, errors.Wrap(e, errors.INTERNAL, "Invalid command payload"))
			return
		}

		out := make(chan error, 1)
		defer close(out)

		go func() {
			cb.Publish(r.Context(), c, out)
		}()

		select {
		case <-r.Context().Done():
			response.RespondJSONError(r.Context(), w, errors.Wrap(r.Context().Err(), errors.TIMEOUT, "Request timeout"))
			return
		case e = <-out:
			if e != nil {
				response.RespondJSONError(r.Context(), w, errors.Wrap(e, errors.INTERNAL, "Command handler error"))
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
			response.RespondJSONError(r.Context(), w, ErrEmptyRequestBody)
			return
		}

		i, _ := identity.FromContext(r.Context())

		user, e := repository.Get(r.Context(), i.ID.String())
		if e != nil {
			response.RespondJSONError(r.Context(), w, errors.Wrap(e, errors.NOTFOUND, "User not found"))
			return
		}

		response.RespondJSON(r.Context(), w, user, http.StatusOK)
		return
	}

	return http.HandlerFunc(fn)
}

// BuildGetUserHandler wraps user gRPC client with http.Handler
func BuildGetUserHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		if r.Body == nil {
			response.RespondJSONError(r.Context(), w, ErrEmptyRequestBody)
			return
		}

		params, ok := context.Parameters(r.Context())
		if !ok {
			response.RespondJSONError(r.Context(), w, ErrInvalidURLParams)
			return
		}

		user, e := repository.Get(r.Context(), params.Value("id"))
		if e != nil {
			response.RespondJSONError(r.Context(), w, errors.Wrap(e, errors.NOTFOUND, "User not found"))
			return
		}

		response.RespondJSON(r.Context(), w, user, http.StatusOK)
		return
	}

	return http.HandlerFunc(fn)
}

// BuildListUserHandler wraps user gRPC client with http.Handler
func BuildListUserHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		if r.Body == nil {
			response.RespondJSONError(r.Context(), w, ErrEmptyRequestBody)
			return
		}

		pageInt, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
		limitInt, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
		page := int32(math.Max(float64(pageInt), 1))
		limit := int32(math.Max(float64(limitInt), 20))

		totalUsers, e := repository.Count(r.Context())
		if e != nil {
			response.RespondJSONError(r.Context(), w, errors.Wrap(e, errors.INTERNAL, "Invalid request"))
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
			response.RespondJSON(r.Context(), w, paginatedList, http.StatusOK)
			return
		}

		paginatedList.Users, e = repository.FindAll(r.Context(), limit, offset)
		if e != nil {
			response.RespondJSONError(r.Context(), w, errors.Wrap(e, errors.INTERNAL, "Invalid request"))
			return
		}

		response.RespondJSON(r.Context(), w, paginatedList, http.StatusOK)
		return
	}

	return http.HandlerFunc(fn)
}
