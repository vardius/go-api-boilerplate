package controller

import (
	"app/pkg/auth"
	"app/pkg/domain"
	"app/pkg/domain/user"
	"app/pkg/middleware"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
)

type authResponse struct {
	AuthToken string         `json:"authToken"`
	Identity  *auth.Identity `json:"identity"`
}

type authCommandPayload struct {
	AuthToken string          `json:"authToken"`
	Data      json.RawMessage `json:"data"`
}

// NewFacebookAuth creates facebook auth handler
func NewFacebookAuth(commandBus domain.CommandBus, jwtService auth.JwtService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.FormValue("accessToken")
		facebookData, err := authCallback(accessToken, "https://graph.facebook.com/me")
		if err != nil {
			r.WithContext(middleware.NewContextWithResponse(r, &middleware.HTTPError{http.StatusBadRequest, err, "Invalid access token"}))
			return
		}

		identity := &auth.Identity{}
		identity.FromFacebookData(facebookData)

		token, err := jwtService.GenerateToken(identity)
		if err != nil {
			r.WithContext(middleware.NewContextWithResponse(r, &middleware.HTTPError{http.StatusInternalServerError, err, "Generate token failure"}))
			return
		}

		out := make(chan error)
		defer close(out)

		go func() {
			commandBus.Publish(user.Domain+user.RegisterWithFacebook, r.Context(), &authCommandPayload{token, facebookData}, out)
		}()

		if err = <-out; err != nil {
			r.WithContext(middleware.NewContextWithResponse(r, &middleware.HTTPError{http.StatusBadRequest, err, "Invalid request"}))
			return
		}

		r.WithContext(middleware.NewContextWithResponse(r, authResponse{token, identity}))
		return
	}
}

// NewGoogleAuth creates google auth handler
func NewGoogleAuth(commandBus domain.CommandBus, jwtService auth.JwtService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.FormValue("accessToken")
		googleData, err := authCallback(accessToken, "https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			r.WithContext(middleware.NewContextWithResponse(r, &middleware.HTTPError{http.StatusBadRequest, err, "Invalid access token"}))
			return
		}

		identity := &auth.Identity{}
		identity.FromGoogleData(googleData)

		token, err := jwtService.GenerateToken(identity)
		if err != nil {
			r.WithContext(middleware.NewContextWithResponse(r, &middleware.HTTPError{http.StatusInternalServerError, err, "Generate token failure"}))
			return
		}

		out := make(chan error)
		defer close(out)

		go func() {
			commandBus.Publish(user.Domain+user.RegisterWithGoogle, r.Context(), &authCommandPayload{token, googleData}, out)
		}()

		if err = <-out; err != nil {
			r.WithContext(middleware.NewContextWithResponse(r, &middleware.HTTPError{http.StatusBadRequest, err, "Invalid request"}))
			return
		}

		r.WithContext(middleware.NewContextWithResponse(r, authResponse{token, identity}))
		return
	}
}

func authCallback(accessToken, apiUrl string) ([]byte, error) {
	resp, err := http.Get(apiUrl + "?access_token=" + url.QueryEscape(accessToken))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
