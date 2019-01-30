package response

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithXSS(t *testing.T) {
	h := WithXSS(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(w, req)

	header := w.Header()
	if header.Get("X-Content-Type-Options") == "" || header.Get("X-Frame-Options") == "" {
		t.Error("WithXSS did not set proper headers")
	}
}
