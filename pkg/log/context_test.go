package log

import (
	"context"
	"testing"
)

func TestContext(t *testing.T) {
	logger := New("development")

	ctx := ContextWithLogger(context.Background(), logger)

	loggerFromContext, ok := FromContext(ctx)
	if !ok || logger != loggerFromContext {
		t.Error("Logger from context did not match the one passed to it")
	}
}
