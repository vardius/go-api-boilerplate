/*
Package firewall allow to guard handlers
*/
package firewall

import (
	"context"

	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// IsGranted returns false if Identity is not set within request's context
// or user does not have required role, true otherwise
func IsGranted(ctx context.Context, role identity.Role) bool {
	i, ok := identity.FromContext(ctx)
	if !ok {
		return false
	}

	return i.HasRole(role)
}
