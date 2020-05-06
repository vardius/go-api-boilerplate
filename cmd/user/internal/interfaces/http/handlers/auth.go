package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
)

type requestBody struct {
	Email string `json:"email"`
}

// BuildSocialAuthHandler wraps user gRPC client with http.Handler
func BuildSocialAuthHandler(apiURL string, cb commandbus.CommandBus, commandName, secretKey string, config oauth2.Config) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.FormValue("accessToken")
		profileData, e := getProfile(accessToken, apiURL)
		if e != nil {
			appErr := errors.Wrap(e, errors.INVALID, "Invalid access token")

			response.MustJSONError(r.Context(), w, appErr)
			return
		}

		c, err := user.NewCommandFromPayload(commandName, profileData)
		if err != nil {
			appErr := errors.Wrap(err, errors.INTERNAL, "Invalid request")

			response.MustJSONError(r.Context(), w, appErr)
			return
		}

		out := make(chan error, 1)
		defer close(out)

		go func() {
			cb.Publish(r.Context(), c, out)
		}()

		select {
		case <-r.Context().Done():
			appErr := errors.Wrap(r.Context().Err(), errors.INTERNAL, "Invalid request")

			response.MustJSONError(r.Context(), w, appErr)
			return
		case err = <-out:
			if err != nil {
				appErr := errors.Wrap(err, errors.INTERNAL, "Invalid request")

				response.MustJSONError(r.Context(), w, appErr)
				return
			}
		}

		emailData := requestBody{}
		e = json.Unmarshal(profileData, &emailData)
		if e != nil {
			appErr := errors.Wrap(e, errors.INTERNAL, "Generate token failure, could not parse body")

			response.MustJSONError(r.Context(), w, appErr)
			return
		}

		token, err := config.PasswordCredentialsToken(r.Context(), emailData.Email, secretKey)
		if err != nil {
			appErr := errors.Wrap(err, errors.INTERNAL, "Generate token failure")

			response.MustJSONError(r.Context(), w, appErr)
			return
		}

		w.WriteHeader(http.StatusOK)
		if err := response.JSON(r.Context(), w, token); err != nil {
			response.MustJSONError(r.Context(), w, errors.Wrap(err, errors.INTERNAL, "Could not parse response"))
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
		return body, errors.Wrap(err, errors.INTERNAL, "Read body error")
	}

	return body, nil
}
