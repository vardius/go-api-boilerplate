package calm_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/vardius/go-api-boilerplate/pkg/common/calm"
	"github.com/vardius/golog"
)

func ExampleRecover() {
	c := calm.New()
	handler := c.RecoverHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("error")
	}))

	// We will mock request for this example
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	handler.ServeHTTP(w, req)

	fmt.Print("I did not break")

	// Output:
	// I did not break
}

func ExampleWithLogger() {
	c := calm.WithLogger(calm.New(), golog.New("debug"))
	handler := c.RecoverHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("error")
	}))

	// We will mock request for this example
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	handler.ServeHTTP(w, req)
}
