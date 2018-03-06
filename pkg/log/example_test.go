package log_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/vardius/go-api-boilerplate/pkg/log"
)

func ExampleLogger() {
	logger := log.New("debug")
	h := logger.LogRequest(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)
}
