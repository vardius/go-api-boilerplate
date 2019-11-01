package handlers

import (
	"net/http"

	"github.com/vardius/go-api-boilerplate/internal/errors"
	"github.com/vardius/go-api-boilerplate/internal/http/response"
	"gopkg.in/oauth2.v3/server"
)

// BuildAuthorizeHandler provides authorize handler
func BuildAuthorizeHandler(srv *server.Server) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		err := srv.HandleAuthorizeRequest(w, r)
		if err != nil {
			response.RespondJSONError(r.Context(), w, errors.New(errors.INVALID, "Authorize request failure"))
			return
		}
	}

	return http.HandlerFunc(fn)
}

// BuildTokenHandler provides token handler
func BuildTokenHandler(srv *server.Server) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		err := srv.HandleTokenRequest(w, r)
		if err != nil {
			response.RespondJSONError(r.Context(), w, errors.New(errors.INTERNAL, "Token request failure"))
			return
		}
	}

	return http.HandlerFunc(fn)
}
