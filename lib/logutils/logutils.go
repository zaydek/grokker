// Package logutils provides logging utilities that configure the default slog
// logger with either JSON or human-readable text output, depending on the provided options.
//
// Usage:
//
//	// Configure the default logger to output human-readable logs.
//	logutils.Configure(logutils.Configuration{IsJSONEnabled: false})
//
//	// Alternatively, configure the logger for JSON formatted output (ideal for structured logging).
//	logutils.Configure(logutils.Configuration{IsJSONEnabled: true})
//
//	// After configuration, use slog for your log messages, for example:
//	//    slog.Info("Logger configured successfully", "mode", "json or text")
package logutils

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

// Configuration is used to configure the default slog logger.
// When IsJSONEnabled is true, the logger outputs logs in JSON format suitable for structured logging.
// When false, the logger uses a text handler (via tint) that produces human-readable logs.
type Configuration struct {
	IsJSONEnabled bool
}

// Configure sets up the package-level default slog logger based on the provided configuration.
//
// The function chooses between two logging handlers based on the IsJSONEnabled flag:
//   - JSON Handler: Uses slog.NewJSONHandler to log in JSON format.
//     Useful for structured logging and machine parsing of log output.
//   - Text Handler: Uses tint.NewHandler to log in a colored, human-friendly text format.
//     Ideal for console output and easier visual inspection.
//
// Both handlers are configured to:
//   - Write logs to os.Stderr.
//   - Include source information (file and line number) via AddSource.
//   - Log messages at the slog.LevelInfo level or higher.
func Configure(config Configuration) {
	if config.IsJSONEnabled {
		// Using JSON handler for structured log output.
		slog.SetDefault(slog.New(
			slog.NewJSONHandler(
				os.Stderr,
				&slog.HandlerOptions{
					AddSource: true,
					Level:     slog.LevelInfo,
				},
			),
		))
	} else {
		// Using tint's text handler for a more readable, console-friendly log output.
		slog.SetDefault(slog.New(
			tint.NewHandler(
				os.Stderr,
				&tint.Options{
					AddSource: true,
					Level:     slog.LevelInfo,
				},
			),
		))
	}
}
