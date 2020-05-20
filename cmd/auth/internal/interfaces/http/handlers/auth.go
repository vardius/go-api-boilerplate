package handlers

import (
	"fmt"
	"net/http"

	"gopkg.in/oauth2.v3/server"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
)

// BuildAuthorizeHandler provides authorize handler
func BuildAuthorizeHandler(srv *server.Server) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if err := srv.HandleAuthorizeRequest(w, r); err != nil {
			appErr := errors.Wrap(fmt.Errorf("%w: Authorize request failure", application.ErrUnauthorized))

			response.MustJSONError(r.Context(), w, appErr)
		}
	}

	return http.HandlerFunc(fn)
}

// BuildTokenHandler provides token handler
func BuildTokenHandler(srv *server.Server) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if err := srv.HandleTokenRequest(w, r); err != nil {
			appErr := errors.Wrap(fmt.Errorf("%w: Token request failure", application.ErrInternal))

			response.MustJSONError(r.Context(), w, appErr)
		}
	}

	return http.HandlerFunc(fn)
}
