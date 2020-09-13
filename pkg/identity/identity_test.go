package identity

import (
	"testing"

	"github.com/google/uuid"
)

func TestNew(t *testing.T) {
	userID := uuid.New()
	userEmail := "example@example.com"
	clientID := uuid.New()
	clientSecret := uuid.New()
	token := "token"

	identity := New(userID, clientID, clientSecret, userEmail, token)

	if identity.UserID != userID {
		t.Errorf("Identity UserID does not match, given: %s | expected %s", identity.UserID, userID)
	}
	if identity.UserEmail != userEmail {
		t.Errorf("Identity UserEmail does not match, given: %s | expected %s", identity.UserEmail, userEmail)
	}
	if identity.ClientID != clientID {
		t.Errorf("Identity ClientID does not match, given: %s | expected %s", identity.ClientID, clientID)
	}
	if identity.ClientSecret.String() != clientSecret.String() {
		t.Errorf("Identity ClientSecret does not match, given: %s | expected %s", identity.ClientSecret, clientSecret)
	}
	if identity.Token != token {
		t.Errorf("Identity Token does not match, given: %s | expected %s", identity.Token, token)
	}
	if !identity.HasRole(RoleUser) {
		t.Errorf("Identity Roles does not match, given: %s | expected %s", identity.Roles, RoleUser)
	}
}

func TestWithToken(t *testing.T) {
	token := "a"
	identity := Identity{
		Token: "b",
	}
	identity.WithToken(token)

	if identity.Token != token {
		t.Errorf("Identity Token does not match, given: %s | expected %s", identity.Token, token)
	}
}

func TestRole_String(t *testing.T) {
	tests := []struct {
		name string
		r    Role
		want string
	}{
		{"USER", RoleUser, "USER"},
		{"ADMIN", RoleAdmin, "ADMIN"},
		{"SUPER_ADMIN", RoleSuperAdmin, "SUPER_ADMIN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
