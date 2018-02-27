package golog

import (
	"context"
)

const DebugLevel = 4
const InfoLevel = 3
const WarningLevel = 2
const ErrorLevel = 1
const CriticalLevel = 0

type (
	Logger interface {
		Debug(ctx context.Context, format string, args ...interface{})
		Info(ctx context.Context, format string, args ...interface{})
		Warning(ctx context.Context, format string, args ...interface{})
		Error(ctx context.Context, format string, args ...interface{})
		Critical(ctx context.Context, format string, args ...interface{})
	}

	loggerFactory func(verbose string) Logger
)

var New loggerFactory

func parseVerboseLevel(verbose string) int {
	switch verbose {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warning":
		return WarningLevel
	case "error":
		return ErrorLevel
	case "critical":
		return CriticalLevel
	default:
		return -1 // logger disabled by default
	}
}
