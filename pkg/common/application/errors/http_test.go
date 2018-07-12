package errors

import (
	"net/http"
	"testing"
)

func TestHTTPStatusCode(t *testing.T) {
	err := New("internal error", INTERNAL)

	if HTTPStatusCode(err) != http.StatusInternalServerError {
		t.Error("Status code does not match")
	}
}
