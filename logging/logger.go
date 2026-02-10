package logging

import (
	"context"
	"log/slog"
	"os"
)

type Logger = *slog.Logger

func Init(ctx context.Context) context.Context {
	var opts = &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	var logger Logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))
	ctx = context.WithValue(ctx, "logger", logger)

	return ctx
}
