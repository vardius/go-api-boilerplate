// +build !appengine,!appenginevm

package golog

import (
	"context"
	"log"
	"os"
)

type (
	fileLogger struct {
		verboseLevel int

		debug    *log.Logger
		info     *log.Logger
		warning  *log.Logger
		error    *log.Logger
		critical *log.Logger
	}
)

const (
	DEBUG = "DEBUG: "
	INFO  = "INFO: "
	WARN  = "WARN: "
	ERROR = "ERROR: "
	FATAL = "FATAL: "
)

func NewFileLogger(verbose, filePath string) Logger {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil
	}

	return &fileLogger{
		parseVerboseLevel(verbose),
		log.New(file, DEBUG, log.Ldate|log.Ltime|log.Lmicroseconds),
		log.New(file, INFO, log.Ldate|log.Ltime|log.Lmicroseconds),
		log.New(file, WARN, log.Ldate|log.Ltime|log.Lmicroseconds),
		log.New(file, ERROR, log.Ldate|log.Ltime|log.Lmicroseconds),
		log.New(file, FATAL, log.Ldate|log.Ltime|log.Lmicroseconds),
	}
}

func (logger *fileLogger) Debug(ctx context.Context, format string, args ...interface{}) {
	if logger.verboseLevel >= DebugLevel {
		logger.debug.Printf(format, args...)
	}
}

func (logger *fileLogger) Info(ctx context.Context, format string, args ...interface{}) {
	if logger.verboseLevel >= InfoLevel {
		logger.info.Printf(format, args...)
	}
}

func (logger *fileLogger) Warning(ctx context.Context, format string, args ...interface{}) {
	if logger.verboseLevel >= WarningLevel {
		logger.warning.Printf(format, args...)
	}
}

func (logger *fileLogger) Error(ctx context.Context, format string, args ...interface{}) {
	if logger.verboseLevel >= ErrorLevel {
		logger.error.Printf(format, args...)
	}
}

func (logger *fileLogger) Critical(ctx context.Context, format string, args ...interface{}) {
	if logger.verboseLevel >= CriticalLevel {
		logger.critical.Printf(format, args...)
	}
}
