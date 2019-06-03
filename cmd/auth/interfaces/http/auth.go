package http

import (
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/gorouter/v4"
	"gopkg.in/oauth2.v3/server"
)

// AddAuthRoutes adds oauth2 routes to router
func AddAuthRoutes(router gorouter.Router, srv *server.Server) {
	router.POST("/authorize", buildAuthorizeHandler(srv))
	router.POST("/token", buildTokenHandler(srv))
}

func buildAuthorizeHandler(srv *server.Server) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		err := srv.HandleAuthorizeRequest(w, r)
		if err != nil {
			response.WithError(r.Context(), errors.Wrap(err, errors.INVALID, "Authorize request failure"))
			return
		}
	}

	return http.HandlerFunc(fn)
}

func buildTokenHandler(srv *server.Server) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		err := srv.HandleTokenRequest(w, r)
		if err != nil {
			response.WithError(r.Context(), errors.Wrap(err, errors.INTERNAL, "Token request failure"))
			return
		}
	}

	return http.HandlerFunc(fn)
}
