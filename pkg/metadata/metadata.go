package metadata

import (
	"context"
	"net"
	"time"

	"github.com/google/uuid"
)

// key represents the type of value for the context key.
type key int

// metadataKey is how request values are stored/retrieved.
const metadataKey key = 1

// Metadata represent state for each request.
type Metadata struct {
	Now        time.Time `json:"-"`
	TraceID    string    `json:"trace_id,omitempty"`
	IPAddress  net.IP    `json:"ip_address,omitempty"`
	StatusCode int       `json:"http_status,omitempty"`
	UserAgent  string    `json:"http_user_agent,omitempty"`
	RemoteAddr string    `json:"http_remote_addr,omitempty"`
	Referer    string    `json:"http_referer,omitempty"`
	Err        error     `json:"-"`
}

func New() *Metadata {
	return &Metadata{
		TraceID: uuid.New().String(),
		Now:     time.Now(),
	}
}

// ContextWithMetadata returns a new Context that carries metadata ptr.
func ContextWithMetadata(ctx context.Context, m *Metadata) context.Context {
	if ctx == nil {
		return nil
	}
	if m == nil {
		return ctx
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
