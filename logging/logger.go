package logging

import (
	"context"
	"log/slog"
	"os"
)

const (
	LevelTrace = slog.LevelInfo - 2
)

type Logger struct {
	logger *slog.Logger
}

func Init(ctx context.Context, level slog.Leveler) (context.Context, Logger) {
	var opts = &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				if level == LevelTrace {
					a.Value = slog.StringValue("TRACE")
				}
			}
			return a
		},
	}

	logger := Logger{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, opts)),
	}

	ctx = context.WithValue(ctx, "logger", logger)

	return ctx, logger
}

func (logger Logger) Error(message string, args ...any) {
	logger.logger.ErrorContext(context.Background(), message, args...)
}

func (logger Logger) ErrorContext(ctx context.Context, message string, args ...any) {
	logger.logger.ErrorContext(ctx, message, args...)
}

func (logger Logger) Warn(message string, args ...any) {
	logger.logger.WarnContext(context.Background(), message, args...)
}

func (logger Logger) WarnContext(ctx context.Context, message string, args ...any) {
	logger.logger.WarnContext(ctx, message, args...)
}

func (logger Logger) Info(message string, args ...any) {
	logger.logger.InfoContext(context.Background(), message, args...)
}

func (logger Logger) InfoContext(ctx context.Context, message string, args ...any) {
	logger.logger.InfoContext(ctx, message, args...)
}

func (logger Logger) Trace(message string, args ...any) {
	logger.TraceContext(context.Background(), message, args...)
}

func (logger Logger) TraceContext(ctx context.Context, message string, args ...any) {
	logger.logger.Log(ctx, LevelTrace, message, args...)
}

func (logger Logger) Debug(message string, args ...any) {
	logger.logger.DebugContext(context.Background(), message, args...)
}

func (logger Logger) DebugContext(ctx context.Context, message string, args ...any) {
	logger.logger.DebugContext(ctx, message, args...)
}
