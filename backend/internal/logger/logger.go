package logger

import (
	"context"
	"log/slog"
	"os"
)

var (
	defaultLogger *slog.Logger
)

// Init инициализирует глобальный логгер
func Init(level string, json bool) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: parseLevel(level),
	}

	if json {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Get возвращает дефолтный логгер
func Get() *slog.Logger {
	if defaultLogger == nil {
		Init("info", false)
	}
	return defaultLogger
}

// WithContext возвращает логгер с контекстными значениями
func WithContext(ctx context.Context) *slog.Logger {
	return Get()
}

// Info логирует на уровне info
func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

// Debug логирует на уровне debug
func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}

// Warn логирует на уровне warn
func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

// Error логирует на уровне error
func Error(msg string, args ...any) {
	Get().Error(msg, args...)
}

// Fatal логирует на уровне error и завершает программу
func Fatal(msg string, args ...any) {
	Get().Error(msg, args...)
	os.Exit(1)
}

// With возвращает логгер с заданными атрибутами
func With(args ...any) *slog.Logger {
	return Get().With(args...)
}