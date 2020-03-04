package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/markbates/goth/gothic"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/internal/commandbus"
	"github.com/vardius/go-api-boilerplate/internal/errors"
	"github.com/vardius/go-api-boilerplate/internal/http/response"
)

type requestBody struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// BuildSocialAuthHandler wraps user gRPC client with http.Handler
func BuildSocialAuthHandler(cb commandbus.CommandBus, commandName string) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// try to get the user without re-authenticating
		gothUser, err := gothic.CompleteUserAuth(w, r)
		if err != nil {
			gothic.BeginAuthHandler(w, r)
		}

		userProfile, err := json.Marshal(gothUser)
		if err != nil {
			response.RespondJSONError(r.Context(), w, errors.Wrap(err, errors.INTERNAL, "Could not json marshal response"))
			return
		}

		c, err := user.NewCommandFromPayload(commandName, userProfile)
		if err != nil {
			response.RespondJSONError(r.Context(), w, errors.Wrap(err, errors.INTERNAL, "Invalid request"))
			return
		}

		out := make(chan error, 1)
		defer close(out)

		go func() {
			cb.Publish(r.Context(), c, out)
		}()

		select {
		case <-r.Context().Done():
			response.RespondJSONError(r.Context(), w, errors.Wrap(r.Context().Err(), errors.INTERNAL, "Invalid request"))
			return
		case err = <-out:
			if err != nil {
				response.RespondJSONError(r.Context(), w, errors.Wrap(err, errors.INTERNAL, "Invalid request"))
				return
			}
		}

		response.RespondJSON(r.Context(), w, gothUser.AccessToken, http.StatusOK)
	}

	return http.HandlerFunc(fn)
}
