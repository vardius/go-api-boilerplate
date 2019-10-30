package response

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

type jsonResponse struct {
	Name string `json:"name"`
}

func TestWithPayloadPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("WithPayload should panic if contextWithResponse was not called first")
		}
	}()

	WithPayload(context.Background(), nil)
}

func TestWithPayload(t *testing.T) {
	ctx := contextWithResponse(context.Background())
	response := "test"

	WithPayload(ctx, response)

	resp, ok := fromContext(ctx)
	if ok && resp.payload == response {
		return
	}

	t.Error("WithPayload failed")
}

func TestWithError(t *testing.T) {
	ctx := contextWithResponse(context.Background())
	respErr := errors.New(errors.INVALID, "Invalid request")

	WithError(ctx, respErr)

	resp, ok := fromContext(ctx)
	if ok && resp.payload == respErr {
		return
	}

	t.Error("WithPayload failed")
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
