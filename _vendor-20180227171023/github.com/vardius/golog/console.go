// +build !appengine,!appenginevm

package golog

import (
	"context"
	"log"
	"os"
)

type (
	consoleLogger struct {
		verboseLevel int

		debug    *log.Logger
		info     *log.Logger
		warning  *log.Logger
		error    *log.Logger
		critical *log.Logger
	}
)

const (
	CLR_0 = "\x1b[30;1m"
	CLR_R = "\x1b[31;1m"
	CLR_G = "\x1b[32;1m"
	CLR_Y = "\x1b[33;1m"
	CLR_B = "\x1b[34;1m"
	CLR_M = "\x1b[35;1m"
	CLR_C = "\x1b[36;1m"
	CLR_W = "\x1b[37;1m"
	CLR_N = "\x1b[0m"
)

func NewConsoleLogger(verbose string) Logger {
	return &consoleLogger{
		parseVerboseLevel(verbose),
		log.New(os.Stdout, CLR_0, log.Ldate|log.Ltime|log.Lmicroseconds),
		log.New(os.Stdout, CLR_G, log.Ldate|log.Ltime|log.Lmicroseconds),
		log.New(os.Stdout, CLR_Y, log.Ldate|log.Ltime|log.Lmicroseconds),
		log.New(os.Stdout, CLR_R, log.Ldate|log.Ltime|log.Lmicroseconds),
		log.New(os.Stdout, CLR_C, log.Ldate|log.Ltime|log.Lmicroseconds),
	}
}

func (logger *consoleLogger) Debug(ctx context.Context, format string, args ...interface{}) {
	if logger.verboseLevel >= DebugLevel {
		logger.debug.Printf("DEBUG: "+format, args...)
	}
}

func (logger *consoleLogger) Info(ctx context.Context, format string, args ...interface{}) {
	if logger.verboseLevel >= InfoLevel {
		logger.info.Printf("INFO:  "+format, args...)
	}
}

func (logger *consoleLogger) Warning(ctx context.Context, format string, args ...interface{}) {
	if logger.verboseLevel >= WarningLevel {
		logger.warning.Printf("WARN:  "+format, args...)
	}
}

func (logger *consoleLogger) Error(ctx context.Context, format string, args ...interface{}) {
	if logger.verboseLevel >= ErrorLevel {
		logger.error.Printf("ERROR: "+format, args...)
	}
}

func (logger *consoleLogger) Critical(ctx context.Context, format string, args ...interface{}) {
	if logger.verboseLevel >= CriticalLevel {
		logger.critical.Printf("FATAL: "+format, args...)
	}
}

func init() {
	New = NewConsoleLogger
}
