package container

import (
	"context"

	"github.com/vardius/gocontainer"
)

// key represents the type of value for the context key.
type key int

// containerKey is how request values are stored/retrieved.
const containerKey key = 1

// ContextWithContainer returns a new Context that carries container ptr.
func ContextWithContainer(ctx context.Context, c gocontainer.Container) context.Context {
	if ctx == nil {
		return nil
	}

	return context.WithValue(ctx, containerKey, c)
}

// FromContext returns the Identity value stored in ctx, if any.
func FromContext(ctx context.Context) (gocontainer.Container, bool) {
	if ctx == nil {
		return nil, false
	}

	m, ok := ctx.Value(containerKey).(gocontainer.Container)

	return m, ok
}
