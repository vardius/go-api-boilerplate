package executioncontext

import "context"

type key struct{}

// Execution context flags
const (
	LIVE   Flag = 1 << iota // live events handling
	REPLAY                  // replay events handling
)

// WithFlag returns a new Context that carries flag.
func WithFlag(ctx context.Context, flag Flag) context.Context {
	flags, ok := ctx.Value(key{}).(Flag)
	if !ok {
		flags = 0
	}

	return context.WithValue(ctx, key{}, flags.Set(flag))
}

// ClearFlag returns a new Context that no longer carries flag.
func ClearFlag(ctx context.Context, flag Flag) context.Context {
	flags, ok := ctx.Value(key{}).(Flag)
	if !ok {
		flags = 0
	}

	return context.WithValue(ctx, key{}, flags.Clear(flag))
}

// ToggleFlag returns a new Context with toggled flag.
func ToggleFlag(ctx context.Context, flag Flag) context.Context {
	flags, ok := ctx.Value(key{}).(Flag)
	if !ok {
		flags = 0
	}

	return context.WithValue(ctx, key{}, flags.Toggle(flag))
}

// FromContext returns the slice of flags stored in ctx
func FromContext(ctx context.Context) Flag {
	flags, ok := ctx.Value(key{}).(Flag)
	if !ok {
		return 0
	}

	return flags
}

// Has returns the bool value based on flag occurrence in context.
func Has(ctx context.Context, flag Flag) bool {
	flags, ok := ctx.Value(key{}).(Flag)
	if !ok {
		return false
	}

	return flags.Has(flag)
}
