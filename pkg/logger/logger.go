package logger

import (
	"io"
	"log/slog"
	"os"
	"sync"
)

// Logger is the interface for structured logging throughout the application
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

// DefaultLogger wraps slog for structured logging
type DefaultLogger struct {
	logger *slog.Logger
}

var (
	defaultLogger *DefaultLogger
	once          sync.Once
)

// Init initializes the global logger
func Init(w io.Writer, level slog.Level) {
	once.Do(func() {
		opts := &slog.HandlerOptions{
			Level: level,
		}
		handler := slog.NewJSONHandler(w, opts)
		defaultLogger = &DefaultLogger{
			logger: slog.New(handler),
		}
	})
}

// Get returns the global logger instance
func Get() Logger {
	if defaultLogger == nil {
		// Initialize with stderr if not explicitly initialized
		Init(os.Stderr, slog.LevelInfo)
	}
	return defaultLogger
}

// Debug logs a debug message
func (l *DefaultLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Info logs an info message
func (l *DefaultLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Warn logs a warning message
func (l *DefaultLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Error logs an error message
func (l *DefaultLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// With returns a new logger with the given attributes
func (l *DefaultLogger) With(args ...any) Logger {
	return &DefaultLogger{
		logger: l.logger.With(args...),
	}
}

// Convenience functions for global logger

// Debug logs a debug message using the global logger
func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}

// Info logs an info message using the global logger
func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

// Warn logs a warning message using the global logger
func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

// Error logs an error message using the global logger
func Error(msg string, args ...any) {
	Get().Error(msg, args...)
}
