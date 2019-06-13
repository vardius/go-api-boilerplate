package http

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/vardius/go-api-boilerplate/cmd/user/domain/user"
	commandbus "github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/gorouter/v4"
	"golang.org/x/oauth2"
)

const googleAPIURL = "https://www.googleapis.com/oauth2/v2/userinfo"
const facebookAPIURL = "https://graph.facebook.com/me"

type requestBody struct {
	Email string `json:"email"`
}

// AddAuthRoutes adds user social media sign-in routes to router
func AddAuthRoutes(router gorouter.Router, cb commandbus.CommandBus, config oauth2.Config, secretKey string) {
	router.POST("/google/callback", buildSocialAuthHandler(googleAPIURL, cb, user.RegisterUserWithGoogle, secretKey, config))
	router.POST("/facebook/callback", buildSocialAuthHandler(facebookAPIURL, cb, user.RegisterUserWithFacebook, secretKey, config))
}

// buildSocialAuthHandler wraps user gRPC client with http.Handler
func buildSocialAuthHandler(apiURL string, cb commandbus.CommandBus, commandName, secretKey string, config oauth2.Config) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.FormValue("accessToken")
		profileData, e := getProfile(accessToken, apiURL)
		if e != nil {
			response.WithError(r.Context(), errors.Wrap(e, errors.INVALID, "Invalid access token"))
			return
		}

		c, err := user.NewCommandFromPayload(commandName, profileData)
		if err != nil {
			response.WithError(r.Context(), errors.Wrap(err, errors.INTERNAL, "Invalid request"))
			return
		}

		out := make(chan error)
		defer close(out)

		go func() {
			cb.Publish(r.Context(), c, out)
		}()

		select {
		case <-r.Context().Done():
			response.WithError(r.Context(), errors.Wrap(r.Context().Err(), errors.INTERNAL, "Invalid request"))
			return
		case err = <-out:
			if e != nil {
				response.WithError(r.Context(), errors.Wrap(err, errors.INTERNAL, "Invalid request"))
				return
			}
		}

		emailData := &requestBody{}
		e = json.Unmarshal(profileData, emailData)
		if e != nil {
			response.WithError(r.Context(), errors.Wrap(e, errors.INTERNAL, "Generate token failure, could not parse body"))
			return
		}

		token, err := config.PasswordCredentialsToken(r.Context(), emailData.Email, secretKey)
		if err != nil {
			response.WithError(r.Context(), errors.Wrap(err, errors.INTERNAL, "Generate token failure"))
			return
		}

		response.WithPayload(r.Context(), token)
	}

	return http.HandlerFunc(fn)
}

func getProfile(accessToken, apiURL string) ([]byte, error) {
	resp, e := http.Get(apiURL + "?access_token=" + url.QueryEscape(accessToken))
	if e != nil {
		return nil, e
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
