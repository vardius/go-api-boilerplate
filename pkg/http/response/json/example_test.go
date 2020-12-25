package json_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response/json"
)

func ExampleJSON() {
	type example struct {
		Name string `json:"name"`
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.JSON(r.Context(), w, http.StatusOK, example{"John"}); err != nil {
			panic(err)
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s", w.Body)

	// Output:
	// {"name":"John"}
}

func ExampleJSON_second() {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.JSON(r.Context(), w, http.StatusOK, nil); err != nil {
			panic(err)
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s", w.Body)

	// Output:
	// {}
}

func ExampleMustJSON() {
	type example struct {
		Name string `json:"name"`
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.MustJSON(r.Context(), w, http.StatusOK, example{"John"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s", w.Body)

	// Output:
	// {"name":"John"}
}

func ExampleMustJSON_second() {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.MustJSON(r.Context(), w, http.StatusOK, nil)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s", w.Body)

	// Output:
	// {}
}

func ExampleJSONError() {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appErr := apperrors.New("response error")

		if err := json.JSONError(r.Context(), w, appErr); err != nil {
			panic(err)
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s", w.Body)

	// Output:
	// {"code":500,"message":"Internal Server Error"}
}

func ExampleMustJSONError() {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appErr := apperrors.New("response error")

		json.MustJSONError(r.Context(), w, appErr)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	h.ServeHTTP(w, req)

	fmt.Printf("%s", w.Body)

	// Output:
	// {"code":500,"message":"Internal Server Error"}
}
