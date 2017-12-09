package socialmedia

import (
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/auth/identity"
	"github.com/vardius/go-api-boilerplate/pkg/auth/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
)

type facebook struct {
	commandBus domain.CommandBus
	jwt        jwt.Jwt
}

func (f *facebook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accessToken := r.FormValue("accessToken")
	data, e := getProfile(accessToken, "https://graph.facebook.com/me")
	if e != nil {
		r.WithContext(response.WithError(r, response.HTTPError{
			Code:    http.StatusBadRequest,
			Error:   e,
			Message: "Invalid access token",
		}))
		return
	}

	identity := &identity.Identity{}
	identity.FromFacebookData(data)

	token, e := f.jwt.Encode(identity)
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
		f.commandBus.Publish(r.Context(), user.RegisterWithFacebook, payload.toJSON(), out)
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

// NewFacebook creates facebook auth handler
func NewFacebook(cb domain.CommandBus, j jwt.Jwt) http.Handler {
	return &facebook{cb, j}
}
