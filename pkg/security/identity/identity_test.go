package identity

import (
	"testing"

	"github.com/google/uuid"
)

func TestNew(t *testing.T) {
	identity, err := New()
	if err != nil {
		t.Errorf("%s", err)
	}

	if identity.ID.String() == "" {
		t.Error("Identity should have ID")
	}
}

func TestFromGoogleData(t *testing.T) {
	identity, err := New()
	if err != nil {
		t.Errorf("%s", err)
	}

	identity.FromGoogleData([]byte(`{"email":"test@email.com"}`))

	if identity.ID.String() == "" {
		t.Error("Identity should have ID")
	}
	if identity.Email == "" {
		t.Errorf("Identity Email does not match")
	}
}

func TestFromFacebookData(t *testing.T) {
	identity, err := New()
	if err != nil {
		t.Errorf("%s", err)
	}

	identity.FromFacebookData([]byte(`{"email":"test@email.com"}`))

	if identity.ID.String() == "" {
		t.Error("Identity should have ID")
	}
	if identity.Email == "" {
		t.Errorf("Identity Email does not match")
	}
}

func TestWithEmail(t *testing.T) {
	email := "test@emai.com"

	identity, err := WithEmail(email)
	if err != nil {
		t.Errorf("%s", err)
	}

	if identity.ID.String() == "" {
		t.Error("Identity should have ID")
	}
	if identity.Email != email {
		t.Errorf("Identity Email does not match, given: %s | expected %s", identity.Email, email)
	}
}

func TestWithValues(t *testing.T) {
	id := uuid.New()
	email := "test@emai.com"
	roles := []string{"user"}

	identity := WithValues(id, email, roles)

	if identity.ID != id {
		t.Errorf("Identity ID does not match, given: %s | expected %s", identity.ID, id)
	}
	if identity.Email != email {
		t.Errorf("Identity Email does not match, given: %s | expected %s", identity.Email, email)
	}
	if len(identity.Roles) != len(roles) && identity.Roles[0] != roles[0] {
		t.Errorf("Identity Roles does not match, given: %s | expected %s", identity.Roles, roles)
	}
}
