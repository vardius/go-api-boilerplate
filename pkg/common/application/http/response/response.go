/*
Package response provides helpers and utils for working with HTTP response
*/
package response

import "context"

type responseKey struct{}

type response struct {
	payload interface{}
}

func (r *response) write(payload interface{}) {
	r.payload = payload
}

func contextWithResponse(ctx context.Context) context.Context {
	return context.WithValue(ctx, responseKey{}, &response{})
}

func fromContext(ctx context.Context) (*response, bool) {
	r, ok := ctx.Value(responseKey{}).(*response)

	return r, ok
}

// WithPayload adds payload to context for response
// Will panic if response middleware wasn't used first
func WithPayload(ctx context.Context, payload interface{}) {
	response, ok := fromContext(ctx)
	if !ok {
		panic("Failed to write payload. Use one of response's package middlewares first")
	}

	response.write(payload)
}

// WithError adds error to context for response
// Will panic if response middleware wasn't used first
func WithError(ctx context.Context, err error) {
	WithPayload(ctx, err)
}
