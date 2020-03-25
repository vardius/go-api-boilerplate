package middleware

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/vardius/go-api-boilerplate/pkg/log"
	md "github.com/vardius/go-api-boilerplate/pkg/metadata"
)

func TestHSTS(t *testing.T) {
	m := HSTS()
	h := m(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(w, req)

	if w.Header().Get("Strict-Transport-Security") == "" {
		t.Error("HSTS did not set proper header")
	}
}

func TestXSS(t *testing.T) {
	m := XSS()
	h := m(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/x", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(w, req)

	header := w.Header()
	if header.Get("X-Content-Type-Options") == "" || header.Get("X-Frame-Options") == "" {
		t.Error("XSS did not set proper headers")
	}
}

func TestLimitRequestBody(t *testing.T) {
	m := LimitRequestBody(10)
	h := m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestRecover(t *testing.T) {
	paniced := false
	defer func() {
		if rcv := recover(); rcv != nil {
			paniced = true
		}
	}()

	m := Recover(log.New("development"))
	handler := m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("error")
	}))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(w, req)

	if paniced == true {
		t.Error("RecoverHandler did not recovered")
	}
}

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	c := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		c <- buf.String()
	}()

	f()

	w.Close()
	os.Stdout = old

	return <-c
}

func TestLogger(t *testing.T) {
	output := captureOutput(func() {
		m := Logger(log.New("development"))
		h := m(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		h.ServeHTTP(w, req)
	})

	if output == "" {
		t.Fail()
	}
}

func TestWithMetadata(t *testing.T) {
	m := WithMetadata()
	h := m(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		v, ok := md.FromContext(req.Context())
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
