package metadata

import (
	"context"
	"time"
)

// key represents the type of value for the context key.
type key int

// metadataKey is how request values are stored/retrieved.
const metadataKey key = 1

// Metadata represent state for each request.
type Metadata struct {
	TraceID    string    `json:"traceId"`
	Now        time.Time `json:"now"`
	StatusCode int       `json:"statusCode"`
}

// ContextWithMetadata returns a new Context that carries metadata ptr.
func ContextWithMetadata(ctx context.Context, m *Metadata) context.Context {
	if ctx == nil {
		return nil
	}

	return context.WithValue(ctx, metadataKey, m)
}

// FromContext returns the Identity value stored in ctx, if any.
func FromContext(ctx context.Context) (*Metadata, bool) {
	if ctx == nil {
		return nil, false
	}

	m, ok := ctx.Value(metadataKey).(*Metadata)

	return m, ok
}
