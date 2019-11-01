package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	commandbus "github.com/vardius/go-api-boilerplate/internal/commandbus"
	"github.com/vardius/go-api-boilerplate/internal/errors"
	"github.com/vardius/go-api-boilerplate/internal/http/response"
	"golang.org/x/oauth2"
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
			response.RespondJSONError(r.Context(), w, errors.Wrap(e, errors.INVALID, "Invalid access token"))
			return
		}

		c, err := user.NewCommandFromPayload(commandName, profileData)
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
			if e != nil {
				response.RespondJSONError(r.Context(), w, errors.Wrap(err, errors.INTERNAL, "Invalid request"))
				return
			}
		}

		emailData := requestBody{}
		e = json.Unmarshal(profileData, &emailData)
		if e != nil {
			response.RespondJSONError(r.Context(), w, errors.Wrap(e, errors.INTERNAL, "Generate token failure, could not parse body"))
			return
		}

		token, err := config.PasswordCredentialsToken(r.Context(), emailData.Email, secretKey)
		if err != nil {
			response.RespondJSONError(r.Context(), w, errors.Wrap(err, errors.INTERNAL, "Generate token failure"))
			return
		}

		response.RespondJSON(r.Context(), w, token, http.StatusOK)
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
