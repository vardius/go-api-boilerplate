package handlers

import (
	"net/http"

	"gopkg.in/oauth2.v3/server"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
)

// BuildAuthorizeHandler provides authorize handler
func BuildAuthorizeHandler(srv *server.Server) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if err := srv.HandleAuthorizeRequest(w, r); err != nil {
			appErr := errors.New(errors.INVALID, "Authorize request failure")
			w.WriteHeader(errors.HTTPStatusCode(appErr))

			if err := response.JSON(r.Context(), w, appErr); err != nil {
				panic(err)
			}
		}
	}

	return http.HandlerFunc(fn)
}

// BuildTokenHandler provides token handler
func BuildTokenHandler(srv *server.Server) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if err := srv.HandleTokenRequest(w, r); err != nil {
			appErr := errors.New(errors.INTERNAL, "Token request failure")
			w.WriteHeader(errors.HTTPStatusCode(appErr))

			if err := response.JSON(r.Context(), w, appErr); err != nil {
				panic(err)
			}
		}
	}

	return http.HandlerFunc(fn)
}
