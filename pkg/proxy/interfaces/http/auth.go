package http

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	auth_proto "github.com/vardius/go-api-boilerplate/pkg/auth/infrastructure/proto"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/errors"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/http/response"
	user_proto "github.com/vardius/go-api-boilerplate/pkg/user/infrastructure/proto"
	user_grpc "github.com/vardius/go-api-boilerplate/pkg/user/interfaces/grpc"
	"github.com/vardius/gorouter"
)

const googleAPIURL = "https://www.googleapis.com/oauth2/v2/userinfo"
const facebookAPIURL = "https://graph.facebook.com/me"

type authTokenResponse struct {
	AuthToken string `json:"authToken"`
}

type requestBody struct {
	Email string `json:"email"`
}

// UnmarshalJSON implements json.Unmarshaler interface
func (b *requestBody) UnmarshalJSON(body []byte) error {
	return json.Unmarshal(body, b)
}

// AddAuthRoutes adds user routes to router
func AddAuthRoutes(router gorouter.Router, uc user_proto.UserClient, ac auth_proto.AuthenticationClient) {
	// Social media auth routes
	router.POST("/auth/google/callback", buildSocialAuthHandler(googleAPIURL, user_grpc.RegisterUserWithGoogle, uc, ac))
	router.POST("/auth/facebook/callback", buildSocialAuthHandler(facebookAPIURL, user_grpc.RegisterUserWithFacebook, uc, ac))
}

// buildSocialAuthHandler wraps user gRPC client with http.Handler
func buildSocialAuthHandler(apiURL string, commandName string, uc user_proto.UserClient, ac auth_proto.AuthenticationClient) http.Handler {
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
