package auth

import (
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/auth/identity"
	"github.com/vardius/gorouter"
)

// Firewall returns Status Unauthorized if Identity not set within request's context
// or user does not have required role
func Firewall(role string) gorouter.MiddlewareFunc {
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

			http.Error(w,
				http.StatusText(http.StatusUnauthorized),
				http.StatusUnauthorized,
			)
		}

		return http.HandlerFunc(fn)
	}
}
