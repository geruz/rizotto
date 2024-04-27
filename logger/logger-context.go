package logger

import (
	"context"
)

type (
	logLevel struct{}
)

func WithCustomLogLevel(ctx context.Context, level LogLevel) context.Context {
	ctx = context.WithValue(ctx, logLevel{}, levelIndex(level))

	return ctx
}

func getLogLevel(ctx context.Context) int {
	if ctx == nil {
		return minLevel
	}

	if level, ok := ctx.Value(logLevel{}).(int); ok {
		return level
	}

	return minLevel
}
