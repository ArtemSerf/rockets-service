package log

import (
	"context"
	"log/slog"
	"os"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func Info(ctx context.Context, message string, kvpairs ...any) {
	logger.InfoContext(ctx, message, kvpairs...)
}

func Error(ctx context.Context, err error, message string, kvpairs ...any) {
	logger.ErrorContext(ctx, message, append(kvpairs, "error", err)...)
}
