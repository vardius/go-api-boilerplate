package auth

import (
	"context"
	"net/http"
)

type key struct{}

func NewContext(req *http.Request, i Identity) context.Context {
	return context.WithValue(req.Context(), key{}, i)
}

//FromContext extracts the request Identity ctx, if present.
func FromContext(ctx context.Context) (Identity, bool) {
	i, ok := ctx.Value(key{}).(Identity)
	return i, ok
}
