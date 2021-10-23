package handlers

import (
	"fmt"
	"net/http"

	"gopkg.in/oauth2.v4/server"

	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	httpjson "github.com/vardius/go-api-boilerplate/pkg/http/response/json"
)

// BuildAuthorizeHandler provides authorize handler
func BuildAuthorizeHandler(srv *server.Server) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		// Implementation example
		// https://github.com/go-oauth2/oauth2/blob/b46cf9f1db6551beb549ad1afe69826b3b2f1abf/example/server/server.go#L62-L82
		if err := srv.HandleAuthorizeRequest(w, r); err != nil {
			return apperrors.Wrap(fmt.Errorf("%w: %v", apperrors.ErrInvalid, err))
		}

		return nil
	}

	return httpjson.HandlerFunc(fn)
}

// BuildTokenHandler provides token handler
func BuildTokenHandler(srv *server.Server) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		if err := srv.HandleTokenRequest(w, r); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return httpjson.HandlerFunc(fn)
}
