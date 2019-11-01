package metadata

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithMetadata(t *testing.T) {
	m := WithMetadata()
	h := m(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(w, req)

	_, ok := req.Context().Value(KeyMetadataValues).(*Metadata)
	if !ok {
		t.Error("WithMetadata did not set proper request metadata")
	}
}
