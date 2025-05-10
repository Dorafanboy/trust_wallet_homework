// Package logger defines a generic logging interface for the application.
package logger

// AppLogger defines the contract for logging in the application.
type AppLogger interface {
	// Debug logs a message at DebugLevel.
	Debug(msg string, args ...any)

	// Info logs a message at InfoLevel.
	Info(msg string, args ...any)

	// Warn logs a message at WarnLevel.
	Warn(msg string, args ...any)

	// Error logs a message at ErrorLevel.
	Error(msg string, args ...any)

	// With returns a new logger with the given key-value pairs added to its context.
	With(args ...any) AppLogger
}
