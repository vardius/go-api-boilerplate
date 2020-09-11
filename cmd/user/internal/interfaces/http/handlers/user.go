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
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// BuildUserCommandDispatchHandler
func BuildUserCommandDispatchHandler(cb commandbus.CommandBus) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		if r.Body == nil {
			response.MustJSONError(r.Context(), w, fmt.Errorf("%w: %v", application.ErrInvalid, ErrEmptyRequestBody))
			return
		}

		params, ok := context.Parameters(r.Context())
		if !ok {
			response.MustJSONError(r.Context(), w, fmt.Errorf("%w: %v", application.ErrInvalid, ErrInvalidURLParams))
			return
		}

		defer r.Body.Close()
		body, e := ioutil.ReadAll(r.Body)
		if e != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(e))
			return
		}

		c, e := user.NewCommandFromPayload(params.Value("command"), body)
		if e != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(e))
			return
		}

		if err := cb.Publish(r.Context(), c); err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
			return
		}

		if err := response.JSON(r.Context(), w, http.StatusCreated, nil); err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
		}
	}

	return http.HandlerFunc(fn)
}

// BuildMeHandler
func BuildMeHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		i, _ := identity.FromContext(r.Context())

		u, err := repository.Get(r.Context(), i.UserID.String())
		if err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
			return
		}

		if err := response.JSON(r.Context(), w, http.StatusOK, u); err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
		}
	}

	return http.HandlerFunc(fn)
}

// BuildGetUserHandler
func BuildGetUserHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		params, ok := context.Parameters(r.Context())
		if !ok {
			response.MustJSONError(r.Context(), w, ErrInvalidURLParams)
			return
		}

		u, err := repository.Get(r.Context(), params.Value("id"))
		if err != nil {
			response.MustJSONError(r.Context(), w, err)
			return
		}

		if err := response.JSON(r.Context(), w, http.StatusOK, u); err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
		}
	}

	return http.HandlerFunc(fn)
}

// BuildListUserHandler
func BuildListUserHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		pageInt, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
		limitInt, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
		page := int32(math.Max(float64(pageInt), 1))
		limit := int32(math.Max(float64(limitInt), 20))

		totalUsers, err := repository.Count(r.Context())
		if err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
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
			if err := response.JSON(r.Context(), w, http.StatusOK, paginatedList); err != nil {
				response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
			}
			return
		}

		paginatedList.Users, err = repository.FindAll(r.Context(), limit, offset)
		if err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
			return
		}

		if err := response.JSON(r.Context(), w, http.StatusOK, paginatedList); err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
		}
	}

	return http.HandlerFunc(fn)
}
