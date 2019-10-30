package response

import (
	"context"
	"testing"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

func TestWithPayloadPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("WithPayload should panic if contextWithResponse was not called first")
		}
	}()

	WithPayload(context.Background(), nil)
}

func TestWithPayload(t *testing.T) {
	ctx := contextWithResponse(context.Background())
	response := "test"

	WithPayload(ctx, response)

	resp, ok := fromContext(ctx)
	if ok && resp.payload == response {
		return
	}

	t.Error("WithPayload failed")
}

func TestWithError(t *testing.T) {
	ctx := contextWithResponse(context.Background())
	respErr := errors.New(errors.INVALID, "Invalid request")

	WithError(ctx, respErr)

	resp, ok := fromContext(ctx)
	if ok && resp.payload == respErr {
		return
	}

	t.Error("WithPayload failed")
}
