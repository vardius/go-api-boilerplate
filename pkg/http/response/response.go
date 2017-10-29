package response

import (
	"context"
	"net/http"
)

type responseKey struct{}

// WithPayload add payload for ersponse
func WithPayload(req *http.Request, payload interface{}) context.Context {
	return context.WithValue(req.Context(), responseKey{}, payload)
}

// WithError add error for ersponse
func WithError(req *http.Request, err HTTPError) context.Context {
	return context.WithValue(req.Context(), responseKey{}, err)
}

func fromContext(ctx context.Context) (interface{}, bool) {
	i := ctx.Value(responseKey{})
	return i, i != nil
}
