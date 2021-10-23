package handlers

import (
	"io/ioutil"
	"math"
	"net/http"
	"strconv"

	"github.com/vardius/gorouter/v4/context"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/client"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	httpjson "github.com/vardius/go-api-boilerplate/pkg/http/response/json"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// BuildClientCommandDispatchHandler dispatches domain command
func BuildClientCommandDispatchHandler(cb commandbus.CommandBus) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		if r.Body == nil {
			return apperrors.Wrap(apperrors.ErrInvalid)
		}

		params, ok := context.Parameters(r.Context())
		if !ok {
			return apperrors.Wrap(apperrors.ErrInvalid)
		}

		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return apperrors.Wrap(err)
		}

		c, err := client.NewCommandFromPayload(params.Value("command"), body)
		if err != nil {
			return apperrors.Wrap(err)
		}

		if err := cb.Publish(r.Context(), c); err != nil {
			return apperrors.Wrap(err)
		}

		if err := httpjson.JSON(r.Context(), w, http.StatusCreated, nil); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return httpjson.HandlerFunc(fn)
}

func BuildGetClientHandler(repository persistence.ClientRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		params, ok := context.Parameters(r.Context())
		if !ok {
			return apperrors.Wrap(apperrors.ErrInvalid)
		}

		c, err := repository.Get(r.Context(), params.Value("clientID"))
		if err != nil {
			return apperrors.Wrap(err)
		}

		if err := httpjson.JSON(r.Context(), w, http.StatusOK, struct {
			Domain string `json:"domain"`
		}{
			Domain: c.GetDomain(),
		}); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return httpjson.HandlerFunc(fn)
}

// BuildListClientsHandler lists client credentials by user ID
func BuildListClientsHandler(repository persistence.ClientRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		i, hasIdentity := identity.FromContext(r.Context())
		if !hasIdentity {
			return apperrors.Wrap(apperrors.ErrUnauthorized)
		}

		pageInt, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
		limitInt, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
		page := int64(math.Max(float64(pageInt), 1))
		limit := int64(math.Max(float64(limitInt), 20))

		totalUsers, err := repository.CountByUserID(r.Context(), i.UserID.String())
		if err != nil {
			return apperrors.Wrap(err)
		}

		offset := (page * limit) - limit

		paginatedList := struct {
			Clients []persistence.Client `json:"clients"`
			Page    int64                `json:"page"`
			Limit   int64                `json:"limit"`
			Total   int64                `json:"total"`
		}{
			Page:  page,
			Limit: limit,
			Total: totalUsers,
		}

		if totalUsers < 1 || offset > (totalUsers-1) {
			if err := httpjson.JSON(r.Context(), w, http.StatusOK, paginatedList); err != nil {
				return apperrors.Wrap(err)
			}
			return nil
		}

		paginatedList.Clients, err = repository.FindAllByUserID(r.Context(), i.UserID.String(), limit, offset)
		if err != nil {
			return apperrors.Wrap(err)
		}

		if err := httpjson.JSON(r.Context(), w, http.StatusOK, paginatedList); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return httpjson.HandlerFunc(fn)
}
