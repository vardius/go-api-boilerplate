package auth

import (
	"app/pkg/auth/identity"
	"app/pkg/auth/jwt"
	"app/pkg/domain"
	"app/pkg/domain/user"
	"app/pkg/http/response"
	"net/http"
)

type google struct {
	commandBus domain.CommandBus
	jwt        jwt.Jwt
}

func (g *google) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accessToken := r.FormValue("accessToken")
	data, e := getProfile(accessToken, "https://www.googleapis.com/oauth2/v2/userinfo")
	if e != nil {
		r.WithContext(response.WithError(r, response.HTTPError{http.StatusBadRequest, e, "Invalid access token"}))
		return
	}

	identity := &identity.Identity{}
	identity.FromGoogleData(data)

	token, e := g.jwt.GenerateToken(identity)
	if e != nil {
		r.WithContext(response.WithError(r, response.HTTPError{http.StatusInternalServerError, e, "Generate token failure"}))
		return
	}

	out := make(chan error)
	defer close(out)

	go func() {
		payload := &commandPayload{token, data}
		g.commandBus.Publish(user.Domain+user.RegisterWithGoogle, r.Context(), payload.toJSON(), out)
	}()

	if e = <-out; e != nil {
		r.WithContext(response.WithError(r, response.HTTPError{http.StatusBadRequest, e, "Invalid request"}))
		return
	}

	r.WithContext(response.WithPayload(r, responsePayload{token, identity}))
	return
}

// NewGoogle creates google auth handler
func NewGoogle(cb domain.CommandBus, j jwt.Jwt) http.Handler {
	return &google{cb, j}
}
