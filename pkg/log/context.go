package log

import (
	"context"
)

// key represents the type of value for the context key.
type key int

// loggerKey is how request values are stored/retrieved.
const loggerKey key = 1

// ContextWithLogger returns a new Context that carries logger ptr.
func ContextWithLogger(ctx context.Context, l *Logger) context.Context {
	if ctx == nil {
		return nil
	}

	return context.WithValue(ctx, loggerKey, l)
}

// FromContext returns the Identity value stored in ctx, if any.
func FromContext(ctx context.Context) (*Logger, bool) {
	if ctx == nil {
		return nil, false
	}

	l, ok := ctx.Value(loggerKey).(*Logger)

	return l, ok
}
