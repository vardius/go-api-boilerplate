package handlers

import (
	"fmt"
	"io"
	"net/http"

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
		body, err := io.ReadAll(r.Body)
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

		i, ok := identity.FromContext(r.Context())
		if !ok {
			return apperrors.Wrap(apperrors.ErrUnauthorized)
		}
		// Users may read only their own record: emails and provider IDs are personal data.
		if params.Value("id") != i.UserID.String() {
			return apperrors.Wrap(fmt.Errorf("%w: users can read only their own record", apperrors.ErrForbidden))
		}

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

// BuildListUserHandler lists the users the caller may see, which is only the caller.
func BuildListUserHandler(repository persistence.UserRepository) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		i, ok := identity.FromContext(r.Context())
		if !ok {
			return apperrors.Wrap(apperrors.ErrUnauthorized)
		}

		u, err := repository.Get(r.Context(), i.UserID.String())
		if err != nil {
			return apperrors.Wrap(err)
		}

		paginatedList := struct {
			Users []userView `json:"users"`
			Page  int64      `json:"page"`
			Limit int64      `json:"limit"`
			Total int64      `json:"total"`
		}{
			Users: []userView{newUserView(u)},
			Page:  1,
			Limit: 1,
			Total: 1,
		}

		if err := httpjson.JSON(r.Context(), w, http.StatusOK, paginatedList); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return httpjson.HandlerFunc(fn)
}
