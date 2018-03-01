package socialmedia

import (
	"net/http"

	"github.com/vardius/go-api-boilerplate/internal/user/client"
	"github.com/vardius/go-api-boilerplate/internal/user/domain"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/security/identity"
)

type google struct {
	client client.UserClient
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
	e = g.client.DispatchAndClose(r.Context(), user.RegisterWithGoogle, payload.toJSON())

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
func NewGoogle(c client.UserClient, j jwt.Jwt) http.Handler {
	return &google{c, j}
}
