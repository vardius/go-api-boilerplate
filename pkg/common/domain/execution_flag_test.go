package domain

import (
	"context"
	"testing"
)

func TestContextWithFlag(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithFlag(ctx, "test")

	if !HasFlag(ctx, "test") {
		t.Fail()
	}
}

func TestHasFlag(t *testing.T) {
	ctx := context.Background()
	if HasFlag(ctx, "test") {
		t.Fail()
	}

	ctx = ContextWithFlag(ctx, "test")

	if !HasFlag(ctx, "test") {
		t.Fail()
	}
}

func TestFlagsFromContext(t *testing.T) {
	ctx := context.Background()
	if len(FlagsFromContext(ctx)) > 0 {
		t.Fail()
	}

	ctx = ContextWithFlag(ctx, "test1")
	ctx = ContextWithFlag(ctx, "test2")

	if len(FlagsFromContext(ctx)) != 2 {
		t.Fail()
	}
}
