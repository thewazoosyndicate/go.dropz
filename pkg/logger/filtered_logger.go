package logger

import (
	"fmt"
	"strings"
)

// FilterPattern defines a pattern to filter out from log messages
type FilterPattern struct {
	Pattern string
	Level   string // "debug", "info", "warn", "error", etc.
}

// FilteredLogger wraps a Logger and filters out specific messages
type FilteredLogger struct {
	logger   Logger
	patterns []FilterPattern
}

// NewFilteredLogger creates a new filtered logger that wraps the provided logger
func NewFilteredLogger(baseLogger Logger, patterns []FilterPattern) *FilteredLogger {
	return &FilteredLogger{
		logger:   baseLogger,
		patterns: patterns,
	}
}

// shouldFilter checks if a message should be filtered out based on patterns
func (l *FilteredLogger) shouldFilter(level, message string) bool {
	for _, pattern := range l.patterns {
		if (pattern.Level == "" || strings.EqualFold(pattern.Level, level)) &&
			strings.Contains(message, pattern.Pattern) {
			return true
		}
	}
	return false
}

// Debug logs a debug message
func (l *FilteredLogger) Debug(args ...interface{}) {
	msg := fmt.Sprint(args...)
	if !l.shouldFilter("debug", msg) {
		l.logger.Debug(args...)
	}
}

// Debugf logs a formatted debug message
func (l *FilteredLogger) Debugf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if !l.shouldFilter("debug", msg) {
		l.logger.Debugf(format, args...)
	}
}

// Info logs an info message
func (l *FilteredLogger) Info(args ...interface{}) {
	msg := fmt.Sprint(args...)
	if !l.shouldFilter("info", msg) {
		l.logger.Info(args...)
	}
}

// Infof logs a formatted info message
func (l *FilteredLogger) Infof(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if !l.shouldFilter("info", msg) {
		l.logger.Infof(format, args...)
	}
}

// Warn logs a warning message
func (l *FilteredLogger) Warn(args ...interface{}) {
	msg := fmt.Sprint(args...)
	if !l.shouldFilter("warn", msg) {
		l.logger.Warn(args...)
	}
}

// Warnf logs a formatted warning message
func (l *FilteredLogger) Warnf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if !l.shouldFilter("warn", msg) {
		l.logger.Warnf(format, args...)
	}
}

// Error logs an error message
func (l *FilteredLogger) Error(args ...interface{}) {
	msg := fmt.Sprint(args...)
	if !l.shouldFilter("error", msg) {
		l.logger.Error(args...)
	}
}

// Errorf logs a formatted error message
func (l *FilteredLogger) Errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if !l.shouldFilter("error", msg) {
		l.logger.Errorf(format, args...)
	}
}

// Fatal logs a fatal message
func (l *FilteredLogger) Fatal(args ...interface{}) {
	msg := fmt.Sprint(args...)
	if !l.shouldFilter("fatal", msg) {
		l.logger.Fatal(args...)
	}
}

// Fatalf logs a formatted fatal message
func (l *FilteredLogger) Fatalf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if !l.shouldFilter("fatal", msg) {
		l.logger.Fatalf(format, args...)
	}
}

// Trace logs a trace message
func (l *FilteredLogger) Trace(args ...interface{}) {
	msg := fmt.Sprint(args...)
	if !l.shouldFilter("trace", msg) {
		l.logger.Trace(args...)
	}
}

// Tracef logs a formatted trace message
func (l *FilteredLogger) Tracef(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if !l.shouldFilter("trace", msg) {
		l.logger.Tracef(format, args...)
	}
}

// WithField returns a logger with a field added to its context
func (l *FilteredLogger) WithField(key string, value interface{}) Logger {
	return &FilteredLogger{
		logger:   l.logger.WithField(key, value),
		patterns: l.patterns,
	}
}

// WithFields returns a logger with fields added to its context
func (l *FilteredLogger) WithFields(fields map[string]interface{}) Logger {
	return &FilteredLogger{
		logger:   l.logger.WithFields(fields),
		patterns: l.patterns,
	}
}

// GetBLEFilteredLogger returns a logger instance configured to filter BLE-specific warnings
func GetBLEFilteredLogger() Logger {
	baseLogger := GetLogger()

	// Define patterns to filter out
	patterns := []FilterPattern{
		// Match both direct warnings and those with [Go-WARN] prefix
		{Pattern: "MapToStruct: invalid field detected *adapter.Adapter1Properties.Connectable", Level: "warn"},
		{Pattern: "MapToStruct: invalid field detected *adapter.Adapter1Properties.PowerState", Level: "warn"},
		{Pattern: "MapToStruct: invalid field detected *adapter.Adapter1Properties.Version", Level: "warn"},
		{Pattern: "MapToStruct: invalid field detected *adapter.Adapter1Properties.Manufacturer", Level: "warn"},
		{Pattern: "MapToStruct: invalid field detected *device.Device1Properties.Bonded", Level: "warn"},

		// Also match with the Go-WARN prefix format that might be used in other parts of the system
		{Pattern: "[Go-WARN] MapToStruct: invalid field detected", Level: "warn"},
		{Pattern: "[WARN] MapToStruct: invalid field detected", Level: "warn"},

		// Match the format used when relayed through another logger
		{Pattern: "WARN: [Go-WARN] MapToStruct: invalid field detected", Level: ""},
		{Pattern: "WARN: [WARN] MapToStruct: invalid field detected", Level: ""},
	}

	return NewFilteredLogger(baseLogger, patterns)
}
