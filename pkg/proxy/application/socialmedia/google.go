package socialmedia

import (
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/common/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/common/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/common/security/identity"
	user_grpc_server "github.com/vardius/go-api-boilerplate/pkg/user/interfaces/grpc"
	user_proto "github.com/vardius/go-api-boilerplate/pkg/user/interfaces/proto"
)

type google struct {
	client user_proto.UserClient
	jwt    jwt.Jwt
}

func (g *google) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accessToken := r.FormValue("accessToken")
	data, e := getProfile(accessToken, "https://www.googleapis.com/oauth2/v2/userinfo")
	if e != nil {
		response.WithError(r.Context(), response.HTTPError{
			Code:    http.StatusBadRequest,
			Error:   e,
			Message: "Invalid access token",
		})
		return
	}

	identity := &identity.Identity{}
	identity.FromGoogleData(data)

	token, e := g.jwt.Encode(identity)
	if e != nil {
		response.WithError(r.Context(), response.HTTPError{
			Code:    http.StatusInternalServerError,
			Error:   e,
			Message: "Generate token failure",
		})
		return
	}

	payload := &commandPayload{token, data}
	_, e = g.client.DispatchCommand(r.Context(), &user_proto.DispatchCommandRequest{
		Name:    user_grpc_server.RegisterUserWithGoogle,
		Payload: payload.toJSON(),
	})

	if e != nil {
		response.WithError(r.Context(), response.HTTPError{
			Code:    http.StatusBadRequest,
			Error:   e,
			Message: "Invalid request",
		})
		return
	}

	response.WithPayload(r.Context(), &responsePayload{token, identity})
	return
}

// NewGoogle creates google auth handler
func NewGoogle(c user_proto.UserClient, j jwt.Jwt) http.Handler {
	return &google{c, j}
}
