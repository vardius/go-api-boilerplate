package middleware

import (
	"fmt"
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/http/firewall"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// GrantAccessFor returns Status Unauthorized if
// Identity is not set within request's context
// or user does not have required role
func GrantAccessFor(role identity.Role) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if !firewall.IsGranted(r.Context(), role) {
				response.MustJSONError(r.Context(), w, fmt.Errorf("%w: could not read role from context", application.ErrUnauthorized))
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
