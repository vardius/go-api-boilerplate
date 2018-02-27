package gorouter

import (
	"context"
	"net/http"
)

type key struct{}

func newContext(req *http.Request, params Params) context.Context {
	return context.WithValue(req.Context(), key{}, params)
}

//FromContext extracts the request Params ctx, if present.
func FromContext(ctx context.Context) (Params, bool) {
	params, ok := ctx.Value(key{}).(Params)
	return params, ok
}
