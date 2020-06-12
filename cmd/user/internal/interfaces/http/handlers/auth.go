package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/auth/oauth2"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
)

type requestBody struct {
	Email string `json:"email"`
}

// BuildSocialAuthHandler wraps user gRPC client with http.Handler
func BuildSocialAuthHandler(apiURL string, cb commandbus.CommandBus, commandName string, tokenProvider oauth2.TokenProvider) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.FormValue("accessToken")
		profileData, e := getProfile(accessToken, apiURL)
		if e != nil {
			appErr := errors.Wrap(fmt.Errorf("%w: %s", application.ErrInvalid, e))

			response.MustJSONError(r.Context(), w, appErr)
			return
		}

		c, err := user.NewCommandFromPayload(commandName, profileData)
		if err != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(err))
			return
		}

		out := make(chan error, 1)
		defer close(out)

		go func() {
			cb.Publish(r.Context(), c, out)
		}()

		select {
		case <-r.Context().Done():
			appErr := errors.Wrap(fmt.Errorf("%w: %s", application.ErrTimeout, r.Context().Err()))

			response.MustJSONError(r.Context(), w, appErr)
			return
		case err = <-out:
			if err != nil {
				response.MustJSONError(r.Context(), w, errors.Wrap(err))
				return
			}
		}

		emailData := requestBody{}
		e = json.Unmarshal(profileData, &emailData)
		if e != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(e))
			return
		}

		token, err := tokenProvider.RetrieveToken(r.Context(), emailData.Email)
		if err != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(err))
			return
		}

		w.WriteHeader(http.StatusOK)
		if err := response.JSON(r.Context(), w, token); err != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(err))
		}
	}

	return http.HandlerFunc(fn)
}

func getProfile(accessToken, apiURL string) ([]byte, error) {
	resp, e := http.Get(apiURL + "?access_token=" + url.QueryEscape(accessToken))
	if e != nil {
		return nil, e
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return body, errors.Wrap(err)
	}

	return body, nil
}
