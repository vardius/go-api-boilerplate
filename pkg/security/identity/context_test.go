package identity

import (
	"context"
	"testing"
)

func TestContext(t *testing.T) {
	identity, _ := New()
	ctx := ContextWithIdentity(context.Background(), identity)
	identityFromContext, ok := FromContext(ctx)
	if ok && identity == identityFromContext {
		return
	}

	t.Error("Identity from context did not match the one passed to it")
}
