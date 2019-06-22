package recovery

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/vardius/golog"
)

func TestRecoverHandler(t *testing.T) {
	paniced := false
	defer func() {
		if rcv := recover(); rcv != nil {
			paniced = true
		}
	}()

	handler := WithRecover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("error")
	}))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(w, req)

	if paniced == true {
		t.Error("RecoverHandler did not recovered")
	}
}

func TestRecoverHandlerWithLogger(t *testing.T) {
	paniced := false
	defer func() {
		if rcv := recover(); rcv != nil {
			paniced = true
		}
	}()

	WithLogger(golog.New(golog.Debug))
	handler := WithRecover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("error")
	}))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(w, req)

	if paniced == true {
		t.Error("RecoverHandler did not recoverd")
	}
}
