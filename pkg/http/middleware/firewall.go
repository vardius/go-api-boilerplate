package middleware

import (
	"fmt"
	"net/http"

	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response/json"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// GrantAccessFor returns Status Unauthorized if
// Identity is not set within request's context
// or user does not have required permission
func GrantAccessFor(permission identity.Permission) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			i, ok := identity.FromContext(r.Context())
			if !ok {
				json.MustJSONError(r.Context(), w, apperrors.Wrap(fmt.Errorf("%w: request is missing identity", apperrors.ErrUnauthorized)))
				return
			}
			if !i.Permission.Has(permission) {
				json.MustJSONError(r.Context(), w, apperrors.Wrap(fmt.Errorf("%w: (%d) missing permission %d", apperrors.ErrForbidden, i.Permission, permission)))
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
