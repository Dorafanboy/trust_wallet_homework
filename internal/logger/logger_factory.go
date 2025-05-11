package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"trust_wallet_homework/internal/config"
)

// NewAppLogger creates a new AppLogger instance with the specified level and output format.
func NewAppLogger(cfg config.LoggerConfig) (AppLogger, error) {
	level, err := toSlogLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("logger setup failed: %w", err)
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var out io.Writer = os.Stdout

	handler, err := toSlogHandler(cfg.Format, out, opts)
	if err != nil {
		return nil, fmt.Errorf("logger setup failed: %w", err)
	}

	slogLogger := slog.New(handler)
	slog.SetDefault(slogLogger)

	return NewSlogAdapter(slogLogger), nil
}

// toSlogLevel converts a config.LogLevel to a slog.Level.
func toSlogLevel(level config.LogLevel) (slog.Level, error) {
	switch level {
	case config.LogLevelDebug:
		return slog.LevelDebug, nil
	case config.LogLevelInfo:
		return slog.LevelInfo, nil
	case config.LogLevelWarn:
		return slog.LevelWarn, nil
	case config.LogLevelError:
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unsupported logger level: %s", level)
	}
}

// toSlogHandler creates a slog.Handler based on the config.LogFormat.
func toSlogHandler(format config.LogFormat, out io.Writer, opts *slog.HandlerOptions) (slog.Handler, error) {
	switch format {
	case config.LogFormatJSON:
		return slog.NewJSONHandler(out, opts), nil
	case config.LogFormatText:
		return slog.NewTextHandler(out, opts), nil
	default:
		return nil, fmt.Errorf("unsupported output format: %s", format)
	}
}
