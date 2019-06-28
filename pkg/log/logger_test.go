package log

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	c := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		c <- buf.String()
	}()

	f()

	w.Close()
	os.Stdout = old

	return <-c
}

func TestNew(t *testing.T) {
	bus := New("development")

	if bus == nil {
		t.Fail()
	}
}

// func TestLogRequest(t *testing.T) {
// 	output := captureOutput(func() {
// 		l := New("development")
// 		h := l.LogRequest(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

// 		w := httptest.NewRecorder()
// 		req, err := http.NewRequest("GET", "/", nil)
// 		if err != nil {
// 			t.Fatal(err)
// 		}

// 		h.ServeHTTP(w, req)
// 	})

// 	if output == "" {
// 		t.Fail()
// 	}
// }
