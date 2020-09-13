package middleware

import (
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// GrantAccessFor returns Status Unauthorized if
// Identity is not set within request's context
// or user does not have required role
func GrantAccessFor(role identity.Role) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			i, ok := identity.FromContext(r.Context())
			if !ok {
				response.MustJSONError(r.Context(), w, apperrors.Wrap(application.ErrUnauthorized))
				return
			}
			if !i.HasRole(role) {
				response.MustJSONError(r.Context(), w, apperrors.Wrap(application.ErrForbidden))
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
