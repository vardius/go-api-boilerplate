package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/vardius/go-api-boilerplate/pkg/http/middleware"
)

func ExampleRecover() {
	m := middleware.Recover()
	handler := m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("error")
	}))

	// We will mock request for this example
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	handler.ServeHTTP(w, req)
}

func ExampleHSTS() {
	m := middleware.HSTS()
	h := m(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s", w.Header().Get("Strict-Transport-Security"))

	// Output:
	// max-age=63072000; includeSubDomains
}

func ExampleXSS() {
	m := middleware.XSS()
	h := m(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s\n", w.Header().Get("X-Content-Type-Options"))
	fmt.Printf("%s", w.Header().Get("X-Frame-Options"))

	// Output:
	// nosniff
	// DENY
}

func ExampleLogger() {
	m := middleware.Logger()
	h := m(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)
}
