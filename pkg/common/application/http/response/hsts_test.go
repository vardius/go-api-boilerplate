package response

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithHSTS(t *testing.T) {
	h := WithHSTS(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(w, req)

	if w.Header().Get("Strict-Transport-Security") == "" {
		t.Error("WithHSTS did not set proper header")
	}
}
