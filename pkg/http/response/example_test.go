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
		response.RespondJSON(r.Context(), w, example{"John"}, http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s\n", w.Body)

	// Output:
	// {"name":"John"}
}

func ExampleRespondJSONError() {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.RespondJSONError(r.Context(), w, errors.New(errors.INTERNAL, "response error"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s\n", w.Body)

	// Output:
	// {"code":"internal","message":"response error"}
}
