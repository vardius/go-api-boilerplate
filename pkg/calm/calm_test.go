package calm

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vardius/golog"
)

func TestNew(t *testing.T) {
	t.Parallel()

	calm := New()

	if calm == nil {
		t.Fail()
	}
}

func TestRecoverHandler(t *testing.T) {
	t.Parallel()

	paniced := false
	defer func() {
		if rcv := recover(); rcv != nil {
			paniced = true
		}
	}()

	calm := New()
	handler := calm.RecoverHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("error")
	}))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(w, req)

	if paniced == true {
		t.Error("RecoverHandler dod not recoverd")
	}
}

func TestRecoverHandlerWithLogger(t *testing.T) {
	t.Parallel()

	paniced := false
	defer func() {
		if rcv := recover(); rcv != nil {
			paniced = true
		}
	}()

	calm := WithLogger(New(), golog.New("debug"))
	handler := calm.RecoverHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("error")
	}))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(w, req)

	if paniced == true {
		t.Error("RecoverHandler dod not recoverd")
	}
}
