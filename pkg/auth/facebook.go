package auth

import (
	"app/pkg/domain"
	"app/pkg/domain/user"
	"app/pkg/err"
	"app/pkg/identity"
	"app/pkg/json"
	"app/pkg/jwt"
	"net/http"
)

// NewFacebookAuth creates facebook auth handler
func NewFacebookAuth(commandBus domain.CommandBus, j jwt.Jwt) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.FormValue("accessToken")
		facebookData, e := authCallback(accessToken, "https://graph.facebook.com/me")
		if e != nil {
			r.WithContext(json.WithResponse(r, &err.HTTPError{http.StatusBadRequest, e, "Invalid access token"}))
			return
		}

		identity := &identity.Identity{}
		identity.FromFacebookData(facebookData)

		token, e := j.GenerateToken(identity)
		if e != nil {
			r.WithContext(json.WithResponse(r, &err.HTTPError{http.StatusInternalServerError, e, "Generate token failure"}))
			return
		}

		out := make(chan error)
		defer close(out)

		go func() {
			payload := &authCommandPayload{token, facebookData}
			commandBus.Publish(user.Domain+user.RegisterWithFacebook, r.Context(), payload.toJSON(), out)
		}()

		if e = <-out; e != nil {
			r.WithContext(json.WithResponse(r, &err.HTTPError{http.StatusBadRequest, e, "Invalid request"}))
			return
		}

		r.WithContext(json.WithResponse(r, authResponse{token, identity}))
		return
	}
}
