package logger

import (
	"io"
	"log/slog"
	"os"
)

var (
	// Logger is the global logger instance
	Logger *slog.Logger

	// Debug indicates if debug logging is enabled
	Debug bool
)

// Init initializes the logger with the appropriate level based on debug flag
func Init(debug bool) {
	Debug = debug

	// Set the log level based on debug flag
	var level slog.Level
	if debug {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	// Create a handler with the appropriate level
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	// Create the logger
	Logger = slog.New(handler)

	// Replace the default logger
	slog.SetDefault(Logger)
}

// SetOutput changes the output destination for the logger
func SetOutput(w io.Writer) {
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	Logger = slog.New(handler)
	slog.SetDefault(Logger)
}
