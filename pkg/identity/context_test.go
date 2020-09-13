package identity

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestContext(t *testing.T) {
	identity := Identity{
		UserID: uuid.New(),
	}
	ctx := ContextWithIdentity(context.Background(), &identity)
	identityFromContext, ok := FromContext(ctx)
	if ok && identity.UserID == identityFromContext.UserID {
		return
	}

	t.Error("Identity from context did not match the one passed to it")
}
