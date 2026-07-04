// Package log provides the framework logger implementation.
// It wraps zerolog and satisfies the contract.Logger interface.
// Console output uses zerolog.ConsoleWriter for colored, human-readable logs.
package log

import (
	"os"

	"github.com/Luoyangan/LQBOT/internal/types"
	"github.com/rs/zerolog"
)

// Logger implements contract.Logger using zerolog.
type Logger struct {
	zerolog.Logger
}

// New creates a new Logger with the given log level and optional color config.
// Output goes to stderr with ANSI color support via zerolog.ConsoleWriter.
func New(level types.LogLevel, _ string) *Logger {
	return NewWithConfig(level, false)
}

// NewWithConfig creates a Logger with explicit color control.
// Set noColor to true to disable ANSI colors (e.g. when redirecting to file).
func NewWithConfig(level types.LogLevel, noColor bool) *Logger {
	output := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "15:04:05",
		NoColor:    noColor,
	}

	zl := zerolog.New(output).
		With().
		Timestamp().
		Logger()

	// Set log level
	switch level {
	case types.LogLevelTrace:
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case types.LogLevelDebug:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case types.LogLevelInfo:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case types.LogLevelWarn:
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case types.LogLevelError:
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	return &Logger{Logger: zl}
}

// Debug logs a debug message with optional key-value pairs.
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	event := l.Logger.Debug()
	addFields(event, keysAndValues)
	event.Msg(msg)
}

// Info logs an info message with optional key-value pairs.
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	event := l.Logger.Info()
	addFields(event, keysAndValues)
	event.Msg(msg)
}

// Warn logs a warning message with optional key-value pairs.
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	event := l.Logger.Warn()
	addFields(event, keysAndValues)
	event.Msg(msg)
}

// Error logs an error message with optional key-value pairs.
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	event := l.Logger.Error()
	addFields(event, keysAndValues)
	event.Msg(msg)
}

// addFields adds key-value pairs to a zerolog event.
func addFields(event *zerolog.Event, keysAndValues []interface{}) {
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		event.Any(key, keysAndValues[i+1])
	}
}

// With creates a child logger with additional fields.
func (l *Logger) With(fields map[string]interface{}) *Logger {
	child := l.Logger.With().Fields(fields).Logger()
	return &Logger{Logger: child}
}
