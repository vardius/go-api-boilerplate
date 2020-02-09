package firewall

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/internal/identity"
)

func TestDoNotGrantAccessFor(t *testing.T) {
	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("Should not get access here")
	})
	h := GrantAccessFor("user")(handler)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	i := identity.Identity{
		ID:    uuid.New(),
		Email: "test@emai.com",
		Roles: []string{"not-user"},
	}
	ctx := identity.ContextWithIdentity(req.Context(), i)

	h.ServeHTTP(w, req.WithContext(ctx))
}

func TestGrantAccessFor(t *testing.T) {
	served := false
	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		served = true
	})
	h := GrantAccessFor("user")(handler)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	i := identity.Identity{
		ID:    uuid.New(),
		Email: "test@emai.com",
		Roles: []string{"user"},
	}
	ctx := identity.ContextWithIdentity(req.Context(), i)

	h.ServeHTTP(w, req.WithContext(ctx))

	if !served {
		t.Error("Should get access to handler")
	}
}
