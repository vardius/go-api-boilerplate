package executioncontext

import "context"

type key struct{}

// WithFlag returns a new Context that carries flag.
func WithFlag(ctx context.Context, flag Flag) context.Context {
	if ctx == nil {
		return nil
	}

	flags, ok := ctx.Value(key{}).(Flag)
	if !ok {
		flags = 0
	}

	return context.WithValue(ctx, key{}, flags.set(flag))
}

// ClearFlag returns a new Context that no longer carries flag.
func ClearFlag(ctx context.Context, flag Flag) context.Context {
	if ctx == nil {
		return nil
	}

	flags, ok := ctx.Value(key{}).(Flag)
	if !ok {
		flags = 0
	}

	return context.WithValue(ctx, key{}, flags.clear(flag))
}

// ToggleFlag returns a new Context with toggled flag.
func ToggleFlag(ctx context.Context, flag Flag) context.Context {
	if ctx == nil {
		return nil
	}

	flags, ok := ctx.Value(key{}).(Flag)
	if !ok {
		flags = 0
	}

	return context.WithValue(ctx, key{}, flags.toggle(flag))
}

// FromContext returns the slice of flags stored in ctx
func FromContext(ctx context.Context) Flag {
	if ctx == nil {
		return 0
	}

	flags, ok := ctx.Value(key{}).(Flag)
	if !ok {
		return 0
	}

	return flags
}

// has returns the bool value based on flag occurrence in context.
func Has(ctx context.Context, flag Flag) bool {
	flags, ok := ctx.Value(key{}).(Flag)
	if !ok {
		return false
	}

	return flags.has(flag)
}
