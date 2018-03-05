/*
Package recover allows to recover from panic
*/
package recover_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	rec "github.com/vardius/go-api-boilerplate/pkg/recover"
	"github.com/vardius/golog"
)

func ExampleRecover_RecoverHandler() {
	r := rec.New()
	handler := r.RecoverHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	r := rec.WithLogger(rec.New(), golog.New("debug"))
	handler := r.RecoverHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
