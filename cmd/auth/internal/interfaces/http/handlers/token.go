package handlers

import (
	"io/ioutil"
	"math"
	"net/http"
	"strconv"

	"github.com/vardius/gorouter/v4/context"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// BuildTokenCommandDispatchHandler dispatches domain command
func BuildTokenCommandDispatchHandler(cb commandbus.CommandBus) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var e error

		if r.Body == nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(application.ErrInvalid))
			return
		}

		params, ok := context.Parameters(r.Context())
		if !ok {
			response.MustJSONError(r.Context(), w, errors.Wrap(application.ErrInvalid))
			return
		}

		defer r.Body.Close()
		body, e := ioutil.ReadAll(r.Body)
		if e != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(e))
			return
		}

		c, e := token.NewCommandFromPayload(params.Value("command"), body)
		if e != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(e))
			return
		}

		if err := cb.Publish(r.Context(), c); err != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(err))
			return
		}

		if err := response.JSON(r.Context(), w, http.StatusCreated, nil); err != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(err))
		}
	}

	return http.HandlerFunc(fn)
}

// BuildListTokensHandler lists auth tokens by client and user IDs
func BuildListTokensHandler(repository persistence.TokenRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		params, ok := context.Parameters(r.Context())
		if !ok {
			response.MustJSONError(r.Context(), w, errors.Wrap(application.ErrInvalid))
			return
		}

		i, hasIdentity := identity.FromContext(r.Context())
		if !hasIdentity {
			response.MustJSONError(r.Context(), w, errors.Wrap(application.ErrUnauthorized))
		}

		clientID := params.Value("clientID")
		pageInt, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
		limitInt, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
		page := int32(math.Max(float64(pageInt), 1))
		limit := int32(math.Max(float64(limitInt), 20))

		totalUsers, err := repository.CountByUserID(r.Context(), clientID, i.UserID.String())
		if err != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(err))
			return
		}

		offset := (page * limit) - limit

		paginatedList := struct {
			Tokens []persistence.Token `json:"tokens"`
			Page   int32               `json:"page"`
			Limit  int32               `json:"limit"`
			Total  int32               `json:"total"`
		}{
			Page:  page,
			Limit: limit,
			Total: totalUsers,
		}

		if totalUsers < 1 || offset > (totalUsers-1) {
			if err := response.JSON(r.Context(), w, http.StatusOK, paginatedList); err != nil {
				response.MustJSONError(r.Context(), w, errors.Wrap(err))
			}
			return
		}

		paginatedList.Tokens, err = repository.FindAllByUserID(r.Context(), clientID, i.UserID.String(), limit, offset)
		if err != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(err))
			return
		}

		if err := response.JSON(r.Context(), w, http.StatusOK, paginatedList); err != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(err))
		}
	}

	return http.HandlerFunc(fn)
}

// BuildListUserAuthTokensHandler lists auth tokens by client and user IDs
func BuildListUserAuthTokensHandler(repository persistence.TokenRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		i, hasIdentity := identity.FromContext(r.Context())
		if !hasIdentity {
			response.MustJSONError(r.Context(), w, errors.Wrap(application.ErrUnauthorized))
		}

		pageInt, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
		limitInt, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
		page := int32(math.Max(float64(pageInt), 1))
		limit := int32(math.Max(float64(limitInt), 20))

		totalUsers, err := repository.CountByUserID(r.Context(), i.ClientID.String(), i.UserID.String())
		if err != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(err))
			return
		}

		offset := (page * limit) - limit

		paginatedList := struct {
			AuthTokens []string `json:"auth_tokens"`
			Page       int32    `json:"page"`
			Limit      int32    `json:"limit"`
			Total      int32    `json:"total"`
		}{
			Page:  page,
			Limit: limit,
			Total: totalUsers,
		}

		if totalUsers < 1 || offset > (totalUsers-1) {
			if err := response.JSON(r.Context(), w, http.StatusOK, paginatedList); err != nil {
				response.MustJSONError(r.Context(), w, errors.Wrap(err))
			}
			return
		}

		tokens, err := repository.FindAllByUserID(r.Context(), i.ClientID.String(), i.UserID.String(), limit, offset)
		if err != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(err))
			return
		}

		for _, token := range tokens {
			paginatedList.AuthTokens = append(paginatedList.AuthTokens, token.GetAccess())
		}

		if err := response.JSON(r.Context(), w, http.StatusOK, paginatedList); err != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(err))
		}
	}

	return http.HandlerFunc(fn)
}
