package errors

import (
	nativeErrors "errors"
	"net/http"
	"testing"
)

func TestHTTPStatusCode(t *testing.T) {
	if HTTPStatusCode(nil) != http.StatusOK {
		t.Error("Status code does not match")
	}

	if HTTPStatusCode(New(INTERNAL, "internal error")) != http.StatusInternalServerError {
		t.Error("Status code does not match")
	}

	if HTTPStatusCode(New(FORBIDDEN, "status map")) != errorCodeToHTTPStatusMap[FORBIDDEN] {
		t.Error("Status code does not match")
	}

	if HTTPStatusCode(Wrap(New(INTERNAL, "status map"), "", "")) != errorCodeToHTTPStatusMap[INTERNAL] {
		t.Error("Status code does not match")
	}

	if HTTPStatusCode(New("unknown_code", "internal error")) != http.StatusInternalServerError {
		t.Error("Status code does not match")
	}

	if HTTPStatusCode(New("123", "code as int")) != 123 {
		t.Error("Status code does not match")
	}

	if HTTPStatusCode(nativeErrors.New("native error")) != http.StatusInternalServerError {
		t.Error("Status code does not match")
	}
}
