package identity

import "context"

type key struct{}

// ContextWithIdentity returns a new Context that carries value i.
func ContextWithIdentity(ctx context.Context, i Identity) context.Context {
	if ctx == nil {
		return nil
	}

	return context.WithValue(ctx, key{}, i)
}

// FromContext returns the Identity value stored in ctx, if any.
func FromContext(ctx context.Context) (Identity, bool) {
	if ctx == nil {
		return Identity{}, false
	}

	i, ok := ctx.Value(key{}).(Identity)

	return i, ok
}
