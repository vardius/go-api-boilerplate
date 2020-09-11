package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	appidentity "github.com/vardius/go-api-boilerplate/cmd/user/internal/application/identity"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/auth/oauth2"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
)

type requestBody struct {
	Email string `json:"email"`
}

// BuildSocialAuthHandler wraps user gRPC client with http.Handler
func BuildSocialAuthHandler(apiURL string, cb commandbus.CommandBus, commandName string, tokenProvider oauth2.TokenProvider, identityProvider appidentity.Provider) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.FormValue("accessToken")
		profileData, err := getProfile(accessToken, apiURL)
		if err != nil {
			appErr := apperrors.Wrap(err)

			response.MustJSONError(r.Context(), w, appErr)
			return
		}

		emailData := requestBody{}
		if err := json.Unmarshal(profileData, &emailData); err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
			return
		}

		c, err := user.NewCommandFromPayload(commandName, profileData)
		if err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
			return
		}

		if err := cb.Publish(r.Context(), c); err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
			return
		}

		// We can do that because command handler acknowledges events when persisting
		// aggregate root so we know that event handlers have executed and data was
		// persisted (see: SaveAndAcknowledge method)
		i, err := identityProvider.GetByUserEmail(r.Context(), emailData.Email, config.Env.App.Domain)
		if err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
			return
		}

		token, err := tokenProvider.RetrievePasswordCredentialsToken(r.Context(), i.ClientID.String(), i.ClientSecret.String(), emailData.Email, oauth2.AllScopes)
		if err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
			return
		}

		if err := response.JSON(r.Context(), w, http.StatusOK, token); err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
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
		return body, apperrors.Wrap(err)
	}

	return body, nil
}
