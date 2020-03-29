package response_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
)

func ExampleJSON() {
	type example struct {
		Name string `json:"name"`
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		if err := response.JSON(r.Context(), w, example{"John"}); err != nil {
			panic(err)
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s\n", w.Body)

	// Output:
	// {"name":"John"}
}

func ExampleJSON_second() {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		if err := response.JSON(r.Context(), w, nil); err != nil {
			panic(err)
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s\n", w.Body)

	// Output:
	// {}
}

func ExampleJSON_third() {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appErr := errors.New(errors.INTERNAL, "response error")
		w.WriteHeader(errors.HTTPStatusCode(appErr))

		if err := response.JSON(r.Context(), w, appErr); err != nil {
			panic(err)
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s\n", w.Body)

	// Output:
	// {"code":"internal","message":"response error"}
}
