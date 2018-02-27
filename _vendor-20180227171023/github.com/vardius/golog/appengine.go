// +build appengine appenginevm

package golog

import (
	"context"

	"google.golang.org/appengine/log"
)

type appengineLogger struct {
	verboseLevel int
}

func NewAppengineLogger(verbose string) Logger {
	return &appengineLogger{parseVerboseLevel(verbose)}
}

func (logger *appengineLogger) Debug(ctx context.Context, format string, args ...interface{}) {
	if logger.verboseLevel >= DebugLevel {
		log.Debugf(ctx, format, args...)
	}
}

func (logger *appengineLogger) Info(ctx context.Context, format string, args ...interface{}) {
	if logger.verboseLevel >= InfoLevel {
		log.Infof(ctx, format, args...)
	}
}

func (logger *appengineLogger) Warning(ctx context.Context, format string, args ...interface{}) {
	if logger.verboseLevel >= WarningLevel {
		log.Warningf(ctx, format, args...)
	}
}

func (logger *appengineLogger) Error(ctx context.Context, format string, args ...interface{}) {
	if logger.verboseLevel >= ErrorLevel {
		log.Errorf(ctx, format, args...)
	}
}

func (logger *appengineLogger) Critical(ctx context.Context, format string, args ...interface{}) {
	if logger.verboseLevel >= CriticalLevel {
		log.Criticalf(ctx, format, args...)
	}
}

func init() {
	New = NewAppengineLogger
}
