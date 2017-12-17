package response

import (
	"errors"
	"net/http"
	"testing"
)

func TestWithPayloadPanic(t *testing.T) {
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("WithPayload should panic if contextWithResponse was not called first")
		}
	}()

	WithPayload(req.Context(), nil)
}

func TestWithPayload(t *testing.T) {
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	req = req.WithContext(contextWithResponse(req.Context()))

	response := "test"

	WithPayload(req.Context(), response)

	resp, ok := fromContext(req.Context())
	if ok && resp.payload == response {
		return
	}

	t.Error("WithPayload faild")
}

func TestWithError(t *testing.T) {
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	req = req.WithContext(contextWithResponse(req.Context()))

	respErr := HTTPError{
		Code:    http.StatusBadRequest,
		Error:   errors.New("response error"),
		Message: "Invalid request",
	}

	WithError(req.Context(), respErr)

	resp, ok := fromContext(req.Context())
	if ok && resp.payload == respErr {
		return
	}

	t.Error("WithPayload faild")
}
