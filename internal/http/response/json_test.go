package response

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vardius/go-api-boilerplate/internal/errors"
)

func TestRespondJSON(t *testing.T) {
	type jsonResponse struct {
		Name string `json:"name"`
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(r.Context(), w, jsonResponse{"John"}, http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(w, req)

	header := w.Header()
	if header.Get("Content-Type") != "application/json" {
		t.Error("RespondJSON did not set proper headers")
	}

	cmp := bytes.Compare(w.Body.Bytes(), append([]byte(`{"name":"John"}`), 10))
	if cmp != 0 {
		t.Errorf("RespondJSON returned wrong body: %s | %d", w.Body.String(), cmp)
	}
}

func TestRespondJSONError(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RespondJSONError(r.Context(), w, errors.New(errors.INVALID, "Invalid request"))
	})

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(w, req)

	header := w.Header()
	if header.Get("Content-Type") != "application/json" {
		t.Error("RespondJSONError did not set proper headers")
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("RespondJSONError error code not handled %d", w.Code)
	}
}

func TestInvalidPayloadAsJSON(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(r.Context(), w, make(chan int), http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(w, req)

	header := w.Header()
	if header.Get("Content-Type") != "application/json" {
		t.Error("RespondJSON did not set proper headers")
	}

	if w.Code != http.StatusInternalServerError {
		t.Errorf("RespondJSON error code not handled %d", w.Code)
	}
}
