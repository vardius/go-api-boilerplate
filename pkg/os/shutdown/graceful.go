package shutdown

import (
	"os"
	"os/signal"
	"syscall"
)

// GracefulStop handles signal and graceful shutdown
func GracefulStop(stop func()) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(
		signalChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	exitChan := make(chan int)
	go func() {
		for {
			s := <-signalChan
			switch s {
			case syscall.SIGHUP: // kill -SIGHUP XXXX
			case syscall.SIGINT: // kill -SIGINT XXXX or Ctrl+c
			case syscall.SIGTERM: // kill -SIGTERM XXXX
			case syscall.SIGQUIT: // kill -SIGQUIT XXXX
				exitChan <- 0
			default:
				exitChan <- 1
			}
		}
	}()

	code := <-exitChan

	stop()

	os.Exit(code)
}
