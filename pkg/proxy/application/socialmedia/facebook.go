package socialmedia

import (
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/common/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/common/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/common/security/identity"
	"github.com/vardius/go-api-boilerplate/pkg/proxy/infrastructure/user/grpc"
	"github.com/vardius/go-api-boilerplate/pkg/user/application"
)

type facebook struct {
	client grpc.UserClient
	jwt    jwt.Jwt
}

func (f *facebook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accessToken := r.FormValue("accessToken")
	data, e := getProfile(accessToken, "https://graph.facebook.com/me")
	if e != nil {
		response.WithError(r.Context(), response.HTTPError{
			Code:    http.StatusBadRequest,
			Error:   e,
			Message: "Invalid access token",
		})
		return
	}

	identity := &identity.Identity{}
	identity.FromFacebookData(data)

	token, e := f.jwt.Encode(identity)
	if e != nil {
		response.WithError(r.Context(), response.HTTPError{
			Code:    http.StatusInternalServerError,
			Error:   e,
			Message: "Generate token failure",
		})
		return
	}

	payload := &commandPayload{token, data}
	e = f.client.DispatchAndClose(r.Context(), application.RegisterUserWithFacebook, payload.toJSON())

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

// NewFacebook creates facebook auth handler
func NewFacebook(c grpc.UserClient, j jwt.Jwt) http.Handler {
	return &facebook{c, j}
}
