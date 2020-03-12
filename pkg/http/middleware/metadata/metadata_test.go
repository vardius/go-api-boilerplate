package metadata

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithMetadata(t *testing.T) {
	m := WithMetadata()
	h := m(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		v, ok := req.Context().Value(KeyMetadataValues).(*Metadata)
		if !ok {
			t.Errorf("WithMetadata did not set proper request metadata %v", v)
		}
	}))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(w, req)
}
