package identity

import (
	"testing"

	"github.com/google/uuid"
)

func TestNew(t *testing.T) {
	userID := uuid.New()
	clientID := uuid.New()
	token := "token"

	identity := &Identity{
		Token:      token,
		Permission: PermissionUserRead,

		UserID:   userID,
		ClientID: clientID,
	}

	if identity.UserID != userID {
		t.Errorf("Identity UserID does not match, given: %s | expected %s", identity.UserID, userID)
	}
	if identity.ClientID != clientID {
		t.Errorf("Identity ClientID does not match, given: %s | expected %s", identity.ClientID, clientID)
	}
	if identity.Token != token {
		t.Errorf("Identity Token does not match, given: %s | expected %s", identity.Token, token)
	}
	if !identity.Permission.Has(PermissionUserRead) {
		t.Errorf("Identity permissions do not match, given: %d | expected %d", identity.Permission, PermissionUserRead)
	}
}
