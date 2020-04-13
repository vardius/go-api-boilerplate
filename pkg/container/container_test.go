package container

import (
	"context"
	"testing"

	"github.com/vardius/gocontainer"
)

func TestContext(t *testing.T) {
	container := gocontainer.New()

	ctx := ContextWithContainer(context.Background(), container)

	containerFromContext, ok := FromContext(ctx)
	if !ok || container != containerFromContext {
		t.Error("Container from context did not match the one passed to it")
	}
}
