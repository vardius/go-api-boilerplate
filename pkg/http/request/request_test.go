package request

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLimitRequestBody(t *testing.T) {
	h := LimitRequestBody(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		_, err := ioutil.ReadAll(r.Body)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", strings.NewReader(`{"name":"John"}`))
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Error("Request body limit")
	}
}
