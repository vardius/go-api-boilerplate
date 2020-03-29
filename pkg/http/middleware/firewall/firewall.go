/*
Package firewall allow to guard handlers
*/
package firewall

import (
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// GrantAccessFor returns Status Unauthorized if
// Identity not set within request's context
// or user does not have required role
func GrantAccessFor(role string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			i, ok := identity.FromContext(r.Context())
			if ok {
				for _, userRole := range i.Roles {
					if userRole == role {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			appErr := errors.New(errors.UNAUTHORIZED, http.StatusText(http.StatusUnauthorized))
			w.WriteHeader(errors.HTTPStatusCode(appErr))

			if err := response.JSON(r.Context(), w, appErr); err != nil {
				panic(err)
			}
		}

		return http.HandlerFunc(fn)
	}
}
