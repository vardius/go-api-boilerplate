package identity

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestContext(t *testing.T) {
	identity := Identity{
		ID: uuid.New(),
	}
	ctx := ContextWithIdentity(context.Background(), identity)
	identityFromContext, ok := FromContext(ctx)
	if ok && identity.ID == identityFromContext.ID {
		return
	}

	t.Error("Identity from context did not match the one passed to it")
}
