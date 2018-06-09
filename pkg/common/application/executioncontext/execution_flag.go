package executioncontext

import "context"

// LIVE execution context flag for live events handling
const LIVE string = "live"

// REPLAY execution context flag for replay events handling
const REPLAY string = "replay"

type key struct{}
type flagMap map[string]bool

// ContextWithFlag returns a new Context that carries flag.
func ContextWithFlag(ctx context.Context, flag string) context.Context {
	flags, ok := ctx.Value(key{}).(flagMap)
	if !ok {
		flags = make(flagMap)
	}

	flags[flag] = true

	return context.WithValue(ctx, key{}, flags)
}

// HasFlag returns the bool value based on flag occurrence in context.
func HasFlag(ctx context.Context, flag string) bool {
	flags, ok := ctx.Value(key{}).(flagMap)
	if !ok {
		return ok
	}

	_, ok = flags[flag]
	return ok
}

// FlagsFromContext returns the slice of flags stored in ctx
func FlagsFromContext(ctx context.Context) []string {
	var emptyFlags []string

	flags, ok := ctx.Value(key{}).(flagMap)

	if !ok {
		return emptyFlags
	}

	fs := make([]string, 0, len(flags))
	for k := range flags {
		fs = append(fs, k)
	}

	return fs
}
