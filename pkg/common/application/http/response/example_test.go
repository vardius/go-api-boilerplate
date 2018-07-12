package response_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/vardius/go-api-boilerplate/pkg/common/application/errors"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/http/response"
)

func ExampleWithHSTS() {
	h := response.WithHSTS(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s\n", w.Header().Get("Strict-Transport-Security"))

	// Output:
	// max-age=63072000; includeSubDomains
}

func ExampleWithXSS() {
	h := response.WithXSS(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s\n", w.Header().Get("X-Content-Type-Options"))
	fmt.Printf("%s\n", w.Header().Get("X-Frame-Options"))

	// Output:
	// nosniff
	// DENY
}

func ExampleAsJSON() {
	type example struct {
		Name string `json:"name"`
	}

	h := response.AsJSON(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		response.WithPayload(r.Context(), example{"John"})
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s\n%s\n", w.Header().Get("Content-Type"), w.Body)

	// Output:
	// application/json
	// {"name":"John"}
}

func ExampleWithPayload() {
	type example struct {
		Name string `json:"name"`
	}

	h := response.AsJSON(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		response.WithPayload(r.Context(), example{"John"})
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s\n", w.Body)

	// Output:
	// {"name":"John"}
}

func ExampleWithError() {
	h := response.AsJSON(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		response.WithError(r.Context(), errors.New("response error", errors.INTERNAL))
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s\n", w.Body)

	// Output:
	// {"code":"internal","message":"response error","error":null}
}
