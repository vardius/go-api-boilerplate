package identity

import (
	"testing"

	"github.com/google/uuid"
)

func TestNew(t *testing.T) {
	id := uuid.New()
	token := "token"
	email := "test@emai.com"

	identity := New(id, token, email)

	if identity.ID != id {
		t.Errorf("Identity ID does not match, given: %s | expected %s", identity.ID, id)
	}
	if identity.Email != email {
		t.Errorf("Identity Email does not match, given: %s | expected %s", identity.Email, email)
	}
	if identity.Token != token {
		t.Errorf("Identity Token does not match, given: %s | expected %s", identity.Token, token)
	}
	if !identity.HasRole(RoleUser) {
		t.Errorf("Identity Roles does not match, given: %s | expected %s", identity.Roles, RoleUser)
	}
}

func TestWithEmail(t *testing.T) {
	email := "test@emai.com"

	identity := Identity{
		Email: "old@email.com",
	}

	newIdentity := identity.WithEmail(email)

	if identity.Email == email {
		t.Error("Identity copy has overridden original instance")
	}

	if newIdentity.Email != email {
		t.Errorf("Identity Email does not match, given: %s | expected %s", identity.Email, email)
	}
}

func TestWithToken(t *testing.T) {
	token := "a"
	identity := Identity{
		Token: "b",
	}
	newIdentity := identity.WithToken(token)

	if identity.Token == token {
		t.Error("Identity copy has overridden original instance")
	}

	if newIdentity.Token != token {
		t.Errorf("Identity Token does not match, given: %s | expected %s", identity.Token, token)
	}
}
