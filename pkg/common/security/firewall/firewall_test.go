package firewall

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/common/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/common/security/identity"
)

func TestDoNotGrantAccessFor(t *testing.T) {
	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("Should not get access here")
	})
	h := response.AsJSON(GrantAccessFor("user")(handler))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.New()
	email := "test@emai.com"
	roles := []string{"not-user"}

	i := identity.WithValues(id, email, roles)
	ctx := identity.ContextWithIdentity(req.Context(), i)

	h.ServeHTTP(w, req.WithContext(ctx))
}

func TestGrantAccessFor(t *testing.T) {
	served := false
	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		served = true
	})
	h := response.AsJSON(GrantAccessFor("user")(handler))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.New()
	email := "test@emai.com"
	roles := []string{"user"}

	i := identity.WithValues(id, email, roles)
	ctx := identity.ContextWithIdentity(req.Context(), i)

	h.ServeHTTP(w, req.WithContext(ctx))

	if !served {
		t.Error("Should get access to handler")
	}
}
