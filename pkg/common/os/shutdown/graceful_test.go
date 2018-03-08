package shutdown

import (
	"syscall"
	"testing"
	"time"
)

func TestGracefulStop(t *testing.T) {
	signals := []syscall.Signal{
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}

	for _, s := range signals {
		done := false
		go func() {
			time.Sleep(10 * time.Millisecond)
			syscall.Kill(syscall.Getpid(), s)
		}()

		GracefulStop(func() {
			done = true
		})
		if !done {
			t.Fatal("Error: syscall.SIGHUP not handled")
		}
	}
}
