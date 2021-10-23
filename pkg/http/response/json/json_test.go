package json

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
)

func TestJSON(t *testing.T) {
	type jsonResponse struct {
		Name string `json:"name"`
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := JSON(r.Context(), w, http.StatusOK, jsonResponse{"John"}); err != nil {
			t.Fatal(err)
		}
	})

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(w, req)

	header := w.Header()
	if header.Get("Content-Type") != "application/json" {
		t.Error("JSON did not set proper headers")
	}

	cmp := bytes.Compare(w.Body.Bytes(), append([]byte(`{"name":"John"}`), 10))
	if cmp != 0 {
		t.Errorf("JSON returned wrong body: %s | %d", w.Body.String(), cmp)
	}
}

func TestJSONNil(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := JSON(r.Context(), w, http.StatusOK, nil); err != nil {
			t.Fatal(err)
		}
	})

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(w, req)

	header := w.Header()
	if header.Get("Content-Type") != "application/json" {
		t.Error("JSON did not set proper headers")
	}

	if w.Code != http.StatusOK {
		t.Errorf("JSON error code does not match %d", w.Code)
	}
}

func TestJSONError(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appErr := apperrors.Wrap(fmt.Errorf("%w: invalid request", apperrors.ErrInvalid))

		if err := JSONError(r.Context(), w, appErr); err != nil {
			t.Fatal(err)
		}
	})

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(w, req)

	header := w.Header()
	if header.Get("Content-Type") != "application/json" {
		t.Error("JSON did not set proper headers")
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("JSON error code not handled %d", w.Code)
	}
}

func TestInvalidPayloadAsJSON(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := JSON(r.Context(), w, http.StatusOK, make(chan int)); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(w, req)

	header := w.Header()
	if header.Get("Content-Type") != "application/json" {
		t.Error("JSON did not set proper headers")
	}

	if w.Code != http.StatusInternalServerError {
		t.Errorf("JSON error code not handled %d", w.Code)
	}
}
