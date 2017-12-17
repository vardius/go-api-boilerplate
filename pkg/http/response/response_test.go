package response

import (
	"errors"
	"net/http"
	"testing"
)

func TestWithPayload(t *testing.T) {
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	req = req.WithContext(contextWithResponse(req.Context()))

	response := "test"

	err = WithPayload(req.Context(), response)
	if err != nil {
		t.Errorf("%s", err)
	}

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

	err = WithError(req.Context(), respErr)
	if err != nil {
		t.Errorf("%s", err)
	}

	resp, ok := fromContext(req.Context())
	if ok && resp.payload == respErr {
		return
	}

	t.Error("WithPayload faild")
}
