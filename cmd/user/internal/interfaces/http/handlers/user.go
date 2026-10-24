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
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	httpjson "github.com/vardius/go-api-boilerplate/pkg/http/response/json"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// dispatchable lists the commands clients may send. Social registration
// commands carry provider identities, so only the provider callbacks issue them.
var dispatchable = map[string]bool{
	user.RegisterUserWithEmail:  true,
	user.RequestUserAccessToken: true,
	user.ChangeUserEmailAddress: true,
}

// userView is what the API shows of a user: never provider access tokens.
type userView struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	FacebookID string `json:"facebook_id,omitempty"`
	GoogleID   string `json:"google_id,omitempty"`
}

func newUserView(u persistence.User) userView {
	return userView{ID: u.GetID(), Email: u.GetEmail(), FacebookID: u.GetFacebookID(), GoogleID: u.GetGoogleID()}
}

// BuildUserCommandDispatchHandler
func BuildUserCommandDispatchHandler(cb commandbus.CommandBus) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		if r.Body == nil {
			return fmt.Errorf("%w: %v", apperrors.ErrInvalid, ErrEmptyRequestBody)
		}

		params, ok := context.Parameters(r.Context())
		if !ok {
			return fmt.Errorf("%w: %v", apperrors.ErrInvalid, ErrInvalidURLParams)
		}

		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return apperrors.Wrap(err)
		}

		contract := params.Value("command")
		if !dispatchable[contract] {
			return fmt.Errorf("%w: command %q can not be dispatched over HTTP", apperrors.ErrInvalid, contract)
		}

		c, err := user.NewCommandFromPayload(contract, body)
		if err != nil {
			return fmt.Errorf("%w: %v", apperrors.ErrInvalid, err)
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

// BuildMeHandler
func BuildMeHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		i, _ := identity.FromContext(r.Context())

		u, err := repository.Get(r.Context(), i.UserID.String())
		if err != nil {
			return apperrors.Wrap(err)
		}

		if err := httpjson.JSON(r.Context(), w, http.StatusOK, newUserView(u)); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return httpjson.HandlerFunc(fn)
}

// BuildGetUserHandler
func BuildGetUserHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		params, ok := context.Parameters(r.Context())
		if !ok {
			return apperrors.Wrap(ErrInvalidURLParams)
		}

		u, err := repository.Get(r.Context(), params.Value("id"))
		if err != nil {
			return apperrors.Wrap(err)
		}

		if err := httpjson.JSON(r.Context(), w, http.StatusOK, newUserView(u)); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return httpjson.HandlerFunc(fn)
}

// BuildListUserHandler
func BuildListUserHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		pageInt, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
		limitInt, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
		page := int64(math.Max(float64(pageInt), 1))
		limit := int64(math.Max(float64(limitInt), 20))

		totalUsers, err := repository.Count(r.Context())
		if err != nil {
			return apperrors.Wrap(err)
		}

		offset := (page * limit) - limit

		paginatedList := struct {
			Users []userView `json:"users"`
			Page  int64      `json:"page"`
			Limit int64      `json:"limit"`
			Total int64      `json:"total"`
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

		users, err := repository.FindAll(r.Context(), limit, offset)
		if err != nil {
			return apperrors.Wrap(err)
		}
		for _, u := range users {
			paginatedList.Users = append(paginatedList.Users, newUserView(u))
		}

		if err := httpjson.JSON(r.Context(), w, http.StatusOK, paginatedList); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return httpjson.HandlerFunc(fn)
}
