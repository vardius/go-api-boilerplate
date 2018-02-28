/*
Package shutdown provides simple shutdown signals handler with callback handler
*/
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
				exitChan <- 0
			case syscall.SIGINT: // kill -SIGINT XXXX or Ctrl+c
				exitChan <- 0
			case syscall.SIGTERM: // kill -SIGTERM XXXX
				exitChan <- 0
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
