package errors

import (
	nativeErrors "errors"
	"net/http"
	"testing"

	appErrors "github.com/vardius/go-api-boilerplate/pkg/errors"
)

func TestHTTPStatusCode(t *testing.T) {
	if HTTPStatusCode(nil) != http.StatusOK {
		t.Error("Status code does not match")
	}

	if HTTPStatusCode(appErrors.New(appErrors.INTERNAL, "internal error")) != http.StatusInternalServerError {
		t.Error("Status code does not match")
	}

	if HTTPStatusCode(appErrors.New(appErrors.FORBIDDEN, "status map")) != errorCodeToHTTPStatusMap[appErrors.FORBIDDEN] {
		t.Error("Status code does not match")
	}

	if HTTPStatusCode(appErrors.Wrap(appErrors.New(appErrors.INTERNAL, "status map"), "", "")) != errorCodeToHTTPStatusMap[appErrors.INTERNAL] {
		t.Error("Status code does not match")
	}

	if HTTPStatusCode(appErrors.New("unknown_code", "internal error")) != http.StatusInternalServerError {
		t.Error("Status code does not match")
	}

	if HTTPStatusCode(appErrors.New("400", "code as int")) != http.StatusBadRequest {
		t.Error("Status code does not match")
	}

	if HTTPStatusCode(nativeErrors.New("native error")) != http.StatusInternalServerError {
		t.Error("Status code does not match")
	}
}
