package logger

import (
	"context"
	"io"
	"log"
)

type ctxKey int

const (
	_ ctxKey = iota
	loggerCtx
)

// FromContext returns the Logger instance from the ctx, or log.Default() if
// none.
func FromContext(ctx context.Context) *log.Logger {
	logger, ok := ctx.Value(loggerCtx).(*log.Logger)
	if ok {
		return logger
	}
	return log.Default()
}

// WithLogger injects an additional logger into ctx.
func WithLogger(ctx context.Context, logger *log.Logger) context.Context {
	return context.WithValue(ctx, loggerCtx, logger)
}

// Silent returns a context.Context with a no-op logger.
func Silent(ctx context.Context) context.Context {
	logger := log.New(io.Discard, "", 0)
	return WithLogger(ctx, logger)
}
