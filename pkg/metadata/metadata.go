package metadata

import (
	"context"
	"time"
)

// ctxMetadataKey represents the type of value for the context key.
type ctxMetadataKey int

// KeyMetadataValues is how request values or stored/retrieved.
const KeyMetadataValues ctxMetadataKey = 1

// Metadata represent state for each request.
type Metadata struct {
	TraceID    string    `json:"traceId"`
	Now        time.Time `json:"now"`
	StatusCode int       `json:"statusCode"`
}

// ContextWithMetadata returns a new Context that carries metadata ptr.
func ContextWithMetadata(ctx context.Context, m *Metadata) context.Context {
	return context.WithValue(ctx, KeyMetadataValues, &m)
}

// FromContext returns the Identity value stored in ctx, if any.
func FromContext(ctx context.Context) (*Metadata, bool) {
	m, ok := ctx.Value(KeyMetadataValues).(*Metadata)

	return m, ok
}
