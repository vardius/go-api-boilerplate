package http

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	auth_proto "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/proto"
	user_proto "github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/proto"
	user_grpc "github.com/vardius/go-api-boilerplate/cmd/user/interfaces/grpc"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/gorouter/v4"
)

const googleAPIURL = "https://www.googleapis.com/oauth2/v2/userinfo"
const facebookAPIURL = "https://graph.facebook.com/me"

type authTokenResponse struct {
	AuthToken string `json:"authToken"`
}

type requestBody struct {
	Email string `json:"email"`
}

// AddAuthRoutes adds user routes to router
func AddAuthRoutes(router gorouter.Router, uc user_proto.UserServiceClient, ac auth_proto.AuthenticationClient) {
	router.POST("/google/callback", buildSocialAuthHandler(googleAPIURL, user_grpc.RegisterUserWithGoogle, uc, ac))
	router.POST("/facebook/callback", buildSocialAuthHandler(facebookAPIURL, user_grpc.RegisterUserWithFacebook, uc, ac))
}

// buildSocialAuthHandler wraps user gRPC client with http.Handler
func buildSocialAuthHandler(apiURL string, commandName string, uc user_proto.UserServiceClient, ac auth_proto.AuthenticationClient) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.FormValue("accessToken")
		profileData, e := getProfile(accessToken, apiURL)
		if e != nil {
			response.WithError(r.Context(), errors.Wrap(e, errors.INVALID, "Invalid access token"))
			return
		}

		_, e = uc.DispatchCommand(r.Context(), &user_proto.DispatchCommandRequest{
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

		tokenResponse, e := ac.GetToken(r.Context(), &auth_proto.GetTokenRequest{
			Email: emailData.Email,
		})

		if e != nil {
			response.WithError(r.Context(), errors.Wrap(e, errors.INTERNAL, "Generate token failure"))
			return
		}

		response.WithPayload(r.Context(), &authTokenResponse{tokenResponse.Token})
		return
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
