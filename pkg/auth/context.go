package auth

import (
	"context"
	"net/http"
)

type key struct{}

// NewContext returns a new Context that carries value i.
func NewContext(req *http.Request, i *Identity) context.Context {
	return context.WithValue(req.Context(), key{}, i)
}

//FromContext returns the Identity value stored in ctx, if any.
func FromContext(ctx context.Context) (*Identity, bool) {
	i, ok := ctx.Value(key{}).(*Identity)
	return i, ok
}
