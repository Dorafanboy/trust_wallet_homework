package logger

import (
	"log/slog"
)

// slogAdapter is a concrete implementation of AppLogger using the standard slog library.
type slogAdapter struct {
	adaptee *slog.Logger
}

// NewSlogAdapter creates a new AppLogger that wraps the given *slog.Logger.
func NewSlogAdapter(slogLogger *slog.Logger) AppLogger {
	if slogLogger == nil {
		slogLogger = slog.Default()
	}
	return &slogAdapter{adaptee: slogLogger}
}

// Debug logs a message at DebugLevel.
func (s *slogAdapter) Debug(msg string, args ...any) {
	s.adaptee.Debug(msg, args...)
}

// Info logs a message at InfoLevel.
func (s *slogAdapter) Info(msg string, args ...any) {
	s.adaptee.Info(msg, args...)
}

// Warn logs a message at WarnLevel.
func (s *slogAdapter) Warn(msg string, args ...any) {
	s.adaptee.Warn(msg, args...)
}

// Error logs a message at ErrorLevel.
func (s *slogAdapter) Error(msg string, args ...any) {
	s.adaptee.Error(msg, args...)
}

// With returns a new AppLogger with the given arguments added to the context.
func (s *slogAdapter) With(args ...any) AppLogger {
	newSlogLogger := s.adaptee.With(args...)
	return &slogAdapter{adaptee: newSlogLogger}
}
