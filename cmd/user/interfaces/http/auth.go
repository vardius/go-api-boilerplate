package http

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	user_proto "github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/proto"
	user_grpc "github.com/vardius/go-api-boilerplate/cmd/user/interfaces/grpc"
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
func AddAuthRoutes(router gorouter.Router, userClient user_proto.UserServiceClient, config oauth2.Config, secretKey string) {
	router.POST("/google/callback", buildSocialAuthHandler(googleAPIURL, user_grpc.RegisterUserWithGoogle, secretKey, config, userClient))
	router.POST("/facebook/callback", buildSocialAuthHandler(facebookAPIURL, user_grpc.RegisterUserWithFacebook, secretKey, config, userClient))
}

// buildSocialAuthHandler wraps user gRPC client with http.Handler
func buildSocialAuthHandler(apiURL, commandName, secretKey string, config oauth2.Config, userClient user_proto.UserServiceClient) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.FormValue("accessToken")
		profileData, e := getProfile(accessToken, apiURL)
		if e != nil {
			response.WithError(r.Context(), errors.Wrap(e, errors.INVALID, "Invalid access token"))
			return
		}

		_, e = userClient.DispatchCommand(r.Context(), &user_proto.DispatchCommandRequest{
			Name:    commandName,
			Payload: profileData,
		})

		if e != nil {
			response.WithError(r.Context(), errors.Wrap(e, errors.INVALID, "Invalid request"))
			return
		}

		emailData := &requestBody{}
		err := json.Unmarshal(profileData, emailData)
		if err != nil {
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
