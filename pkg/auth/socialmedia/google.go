package socialmedia

import (
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/auth/identity"
	"github.com/vardius/go-api-boilerplate/pkg/auth/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
)

type google struct {
	commandBus domain.CommandBus
	jwt        jwt.Jwt
}

func (g *google) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accessToken := r.FormValue("accessToken")
	data, e := getProfile(accessToken, "https://www.googleapis.com/oauth2/v2/userinfo")
	if e != nil {
		r.WithContext(response.WithError(r, response.HTTPError{
			Code:    http.StatusBadRequest,
			Error:   e,
			Message: "Invalid access token",
		}))
		return
	}

	identity := &identity.Identity{}
	identity.FromGoogleData(data)

	token, e := g.jwt.Encode(identity)
	if e != nil {
		r.WithContext(response.WithError(r, response.HTTPError{
			Code:    http.StatusInternalServerError,
			Error:   e,
			Message: "Generate token failure",
		}))
		return
	}

	out := make(chan error)
	defer close(out)

	go func() {
		payload := &commandPayload{token, data}
		g.commandBus.Publish(r.Context(), user.Domain+user.RegisterWithGoogle, payload.toJSON(), out)
	}()

	if e = <-out; e != nil {
		r.WithContext(response.WithError(r, response.HTTPError{
			Code:    http.StatusBadRequest,
			Error:   e,
			Message: "Invalid request",
		}))
		return
	}

	r.WithContext(response.WithPayload(r, &responsePayload{token, identity}))
	return
}

// NewGoogle creates google auth handler
func NewGoogle(cb domain.CommandBus, j jwt.Jwt) http.Handler {
	return &google{cb, j}
}
