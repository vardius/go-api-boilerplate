package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"

	appidentity "github.com/vardius/go-api-boilerplate/cmd/user/internal/application/identity"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	auth "github.com/vardius/go-api-boilerplate/pkg/auth/oauth2"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
)

type requestBody struct {
	Email string `json:"email"`
}

const authCookieName = "oauthstate"

// BuildSocialAuthHandler wraps user gRPC client with http.Handler
func BuildSocialAuthHandler(config *oauth2.Config) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		expiration := time.Now().Add(365 * 24 * time.Hour)

		b := make([]byte, 16)
		if _, err := io.ReadFull(rand.Reader, b); err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
			return
		}

		state := base64.URLEncoding.EncodeToString(b)

		cookie := http.Cookie{Name: authCookieName, Value: state, Expires: expiration}
		http.SetCookie(w, &cookie)

		http.Redirect(w, r, config.AuthCodeURL(state), http.StatusTemporaryRedirect)
	}

	return http.HandlerFunc(fn)
}

// BuildAuthCallbackHandler wraps user gRPC client with http.Handler
func BuildAuthCallbackHandler(authConfig *oauth2.Config, apiURL string, cb commandbus.CommandBus, commandName string, tokenProvider auth.TokenProvider, identityProvider appidentity.Provider) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		oauthState, _ := r.Cookie(authCookieName)
		if r.FormValue("state") != oauthState.Value {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(fmt.Errorf("invalid oauth state")))
			return
		}

		oauthToken, err := authConfig.Exchange(r.Context(), r.FormValue("code"))
		if err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
			return
		}

		profileData, err := getProfile(oauthToken.AccessToken, apiURL)
		if err != nil {
			appErr := apperrors.Wrap(err)

			response.MustJSONError(r.Context(), w, appErr)
			return
		}

		var emailData requestBody
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
		i, err := identityProvider.GetByUserEmail(r.Context(), emailData.Email)
		if err != nil {
			response.MustJSONError(r.Context(), w, apperrors.Wrap(err))
			return
		}

		token, err := tokenProvider.RetrievePasswordCredentialsToken(r.Context(), i.ClientID.String(), i.ClientSecret.String(), emailData.Email, auth.AllScopes)
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
