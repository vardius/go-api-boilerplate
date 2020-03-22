package response_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
)

func ExampleRespondJSON() {
	type example struct {
		Name string `json:"name"`
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.WriteHeader(r.Context(), w, http.StatusOK)

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

func ExampleRespondJSON_second() {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.WriteHeader(r.Context(), w, http.StatusOK)

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

func ExampleRespondJSONError() {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appErr := errors.New(errors.INTERNAL, "response error")
		response.WriteHeader(r.Context(), w, errors.HTTPStatusCode(appErr))

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
