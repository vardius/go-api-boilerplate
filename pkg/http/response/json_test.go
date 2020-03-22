package response

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

func TestRespondJSON(t *testing.T) {
	type jsonResponse struct {
		Name string `json:"name"`
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteHeader(r.Context(), w, http.StatusOK)

		if err := JSON(r.Context(), w, jsonResponse{"John"}); err != nil {
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
		t.Error("RespondJSON did not set proper headers")
	}

	cmp := bytes.Compare(w.Body.Bytes(), append([]byte(`{"name":"John"}`), 10))
	if cmp != 0 {
		t.Errorf("RespondJSON returned wrong body: %s | %d", w.Body.String(), cmp)
	}
}

func TestRespondJSONNil(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := JSON(r.Context(), w, nil); err != nil {
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
		t.Error("RespondJSON did not set proper headers")
	}

	if w.Code != http.StatusNoContent {
		t.Errorf("RespondJSON error code does not match %d", w.Code)
	}
}

func TestRespondJSONNilWithStatusOk(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteHeader(r.Context(), w, http.StatusOK)

		if err := JSON(r.Context(), w, nil); err != nil {
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
		t.Error("RespondJSON did not set proper headers")
	}

	if w.Code != http.StatusOK {
		t.Errorf("RespondJSON error code does not match %d", w.Code)
	}
}

func TestRespondJSONError(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appErr := errors.New(errors.INVALID, "Invalid request")

		WriteHeader(r.Context(), w, errors.HTTPStatusCode(appErr))

		if err := JSON(r.Context(), w, appErr); err != nil {
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
		t.Error("RespondJSONError did not set proper headers")
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("RespondJSONError error code not handled %d", w.Code)
	}
}

func TestInvalidPayloadAsJSON(t *testing.T) {
	paniced := false
	defer func() {
		if rcv := recover(); rcv != nil {
			paniced = true
		}
	}()

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteHeader(r.Context(), w, http.StatusOK)

		if err := JSON(r.Context(), w, make(chan int)); err != nil {
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
		t.Error("RespondJSON did not set proper headers")
	}

	if paniced == true {
		t.Error("Did not panic")
	}

	if w.Code != http.StatusInternalServerError {
		t.Errorf("RespondJSON error code not handled %d", w.Code)
	}
}
