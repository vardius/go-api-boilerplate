package log_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/vardius/go-api-boilerplate/pkg/common/log"
)

func ExampleLogger_LogRequest() {
	l := log.New("development")
	h := l.LogRequest("testserver")(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)
}
