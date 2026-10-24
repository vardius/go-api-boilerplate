package handlers

import (
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/vardius/gorouter/v4/context"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	httpjson "github.com/vardius/go-api-boilerplate/pkg/http/response/json"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// BuildTokenCommandDispatchHandler dispatches domain command
func BuildTokenCommandDispatchHandler(cb commandbus.CommandBus) http.Handler {
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

		c, err := token.NewCommandFromPayload(params.Value("command"), body)
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

// BuildListTokensHandler lists auth tokens by client and user IDs
func BuildListTokensHandler(repository persistence.TokenRepository, clientRepository persistence.ClientRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		params, ok := context.Parameters(r.Context())
		if !ok {
			return apperrors.Wrap(apperrors.ErrInvalid)
		}

		clientID := params.Value("clientID")

		i, hasIdentity := identity.FromContext(r.Context())
		if !hasIdentity {
			return apperrors.Wrap(apperrors.ErrUnauthorized)
		}

		c, err := clientRepository.Get(r.Context(), clientID)
		if err != nil {
			return apperrors.Wrap(err)
		}
		if c.GetUserID() != i.UserID.String() {
			return apperrors.Wrap(apperrors.ErrForbidden)
		}

		pageInt, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
		limitInt, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
		page := int64(math.Max(float64(pageInt), 1))
		limit := int64(math.Max(float64(limitInt), 20))

		totalUsers, err := repository.CountByClientID(r.Context(), clientID)
		if err != nil {
			return apperrors.Wrap(err)
		}

		offset := (page * limit) - limit

		paginatedList := struct {
			Tokens []tokenView `json:"tokens"`
			Page   int64       `json:"page"`
			Limit  int64       `json:"limit"`
			Total  int64       `json:"total"`
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

		tokens, err := repository.FindAllByClientID(r.Context(), clientID, limit, offset)
		if err != nil {
			return apperrors.Wrap(err)
		}
		paginatedList.Tokens, err = newTokenViews(tokens)
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

// BuildListUserAuthTokensHandler lists the signed-in user's own login tokens.
func BuildListUserAuthTokensHandler(repository persistence.TokenRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		i, hasIdentity := identity.FromContext(r.Context())
		if !hasIdentity {
			return apperrors.Wrap(apperrors.ErrUnauthorized)
		}

		params, ok := context.Parameters(r.Context())
		if !ok {
			return apperrors.Wrap(apperrors.ErrInvalid)
		}
		if params.Value("userID") != i.UserID.String() {
			return apperrors.Wrap(fmt.Errorf("%w: users can only list their own tokens", apperrors.ErrForbidden))
		}

		pageInt, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
		limitInt, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
		page := int64(math.Max(float64(pageInt), 1))
		limit := int64(math.Max(float64(limitInt), 20))

		userID := i.UserID.String()
		loginClientID := uuid.Nil.String()

		total, err := repository.CountByUserID(r.Context(), userID, loginClientID)
		if err != nil {
			return apperrors.Wrap(err)
		}

		offset := (page * limit) - limit

		paginatedList := struct {
			AuthTokens []tokenView `json:"auth_tokens"`
			Page       int64       `json:"page"`
			Limit      int64       `json:"limit"`
			Total      int64       `json:"total"`
		}{
			Page:  page,
			Limit: limit,
			Total: total,
		}

		if total < 1 || offset > (total-1) {
			if err := httpjson.JSON(r.Context(), w, http.StatusOK, paginatedList); err != nil {
				return apperrors.Wrap(err)
			}
			return nil
		}

		tokens, err := repository.FindAllByUserID(r.Context(), userID, loginClientID, limit, offset)
		if err != nil {
			return apperrors.Wrap(err)
		}
		paginatedList.AuthTokens, err = newTokenViews(tokens)
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

// tokenView is what the API shows of a token: never the access or refresh token itself.
type tokenView struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	UserAgent string    `json:"user_agent,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
}

func newTokenViews(tokens []persistence.Token) ([]tokenView, error) {
	views := make([]tokenView, 0, len(tokens))
	for _, t := range tokens {
		ti, err := t.TokenInfo()
		if err != nil {
			return nil, err
		}
		views = append(views, tokenView{
			ID:        t.GetID(),
			ClientID:  ti.GetClientID(),
			UserAgent: t.GetUserAgent(),
			ExpiresAt: ti.GetAccessCreateAt().Add(ti.GetAccessExpiresIn()),
		})
	}
	return views, nil
}
