package handlers

import (
	"io/ioutil"
	"math"
	"net/http"
	"strconv"

	"github.com/vardius/gorouter/v4/context"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/client"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// BuildClientCommandDispatchHandler dispatches domain command
func BuildClientCommandDispatchHandler(cb commandbus.CommandBus) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		if r.Body == nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(application.ErrInvalid))
			return
		}

		params, ok := context.Parameters(r.Context())
		if !ok {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(application.ErrInvalid))
			return
		}

		defer r.Body.Close()
		body, e := ioutil.ReadAll(r.Body)
		if e != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(e))
			return
		}

		c, e := client.NewCommandFromPayload(params.Value("command"), body)
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

func BuildGetClientHandler(repository persistence.ClientRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		params, ok := context.Parameters(r.Context())
		if !ok {
			response.MustJSONError(r.Context(), w, application.ErrInvalid)
			return
		}

		c, err := repository.Get(r.Context(), params.Value("clientID"))
		if err != nil {
			response.MustJSONError(r.Context(), w, err)
			return
		}

		if err := response.JSON(r.Context(), w, http.StatusOK, struct {
			Domain string `json:"domain"`
		}{
			Domain: c.GetDomain(),
		}); err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
		}
	}

	return http.HandlerFunc(fn)
}

// BuildListClientsHandler lists client credentials by user ID
func BuildListClientsHandler(repository persistence.ClientRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		i, hasIdentity := identity.FromContext(r.Context())
		if !hasIdentity {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(application.ErrUnauthorized))
		}

		pageInt, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
		limitInt, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
		page := int32(math.Max(float64(pageInt), 1))
		limit := int32(math.Max(float64(limitInt), 20))

		totalUsers, err := repository.CountByUserID(r.Context(), i.UserID.String())
		if err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
			return
		}

		offset := (page * limit) - limit

		paginatedList := struct {
			Clients []persistence.Client `json:"clients"`
			Page    int32                `json:"page"`
			Limit   int32                `json:"limit"`
			Total   int32                `json:"total"`
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

		paginatedList.Clients, err = repository.FindAllByUserID(r.Context(), i.UserID.String(), limit, offset)
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
