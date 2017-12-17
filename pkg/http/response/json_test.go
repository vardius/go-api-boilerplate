package response

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type jsonResponse struct {
	Name string `json:"name"`
}

func TestAsJSON(t *testing.T) {
	h := AsJSON(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		WithPayload(r.Context(), jsonResponse{"John"})
	}))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(w, req)

	header := w.Header()
	if header.Get("Content-Type") != "application/json" {
		t.Error("AsJSON did not set proper headers")
	}

	cmp := bytes.Compare(w.Body.Bytes(), append([]byte(`{"name":"John"}`), 10))
	if cmp != 0 {
		t.Errorf("AsJSON returned wrong body: %s | %d", w.Body.String(), cmp)
	}
}

func TestErrorAsJSON(t *testing.T) {
	h := AsJSON(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		WithPayload(r.Context(), HTTPError{
			Code:    http.StatusBadRequest,
			Error:   errors.New("response error"),
			Message: "Invalid request",
		})
	}))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(w, req)

	header := w.Header()
	if header.Get("Content-Type") != "application/json" {
		t.Error("AsJSON did not set proper headers")
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("AsJSON error code not handled %d", w.Code)
	}
}

func TestErrorPointerAsJSON(t *testing.T) {
	h := AsJSON(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		WithPayload(r.Context(), &HTTPError{
			Code:    http.StatusBadRequest,
			Error:   errors.New("response error"),
			Message: "Invalid request",
		})
	}))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(w, req)

	header := w.Header()
	if header.Get("Content-Type") != "application/json" {
		t.Error("AsJSON did not set proper headers")
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("AsJSON error code not handled %d", w.Code)
	}
}
