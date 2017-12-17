package response

import (
	"context"
	"errors"
)

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
func WithPayload(ctx context.Context, payload interface{}) error {
	response, ok := fromContext(ctx)
	if !ok {
		return errors.New("Faild to write payload")
	}

	response.write(payload)

	return nil
}

// WithError adds error to context for response
func WithError(ctx context.Context, err HTTPError) error {
	e := WithPayload(ctx, err)
	if e != nil {
		return errors.New("Faild to write error")
	}

	return nil
}
