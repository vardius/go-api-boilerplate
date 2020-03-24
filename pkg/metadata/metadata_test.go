package metadata

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestContext(t *testing.T) {
	m := Metadata{
		TraceID: uuid.New().String(),
		Now:     time.Now(),
	}
	ctx := ContextWithMetadata(context.Background(), &m)
	identityFromContext, ok := FromContext(ctx)
	if ok && m.TraceID == identityFromContext.TraceID {
		return
	}

	t.Error("Metadata from context did not match the one passed to it")
}
