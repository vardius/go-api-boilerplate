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
	logger := New("development")

	if logger == nil {
		t.Fail()
	}
}
