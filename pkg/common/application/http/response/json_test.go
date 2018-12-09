package response

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vardius/go-api-boilerplate/pkg/common/application/errors"
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
		WithError(r.Context(), errors.New(errors.INVALID, "Invalid request"))
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

func TestErrorPayloadAsJSON(t *testing.T) {
	h := AsJSON(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		WithPayload(r.Context(), errors.New(errors.INVALID, "Invalid request"))
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

func TestInvalidPayloadAsJSON(t *testing.T) {
	h := AsJSON(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		WithPayload(r.Context(), make(chan int))
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

	if w.Code != http.StatusInternalServerError {
		t.Errorf("AsJSON error code not handled %d", w.Code)
	}
}
