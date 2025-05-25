package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
)

// Logger is the interface for logging operations
type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Trace(args ...interface{})
	Tracef(format string, args ...interface{})
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
}

// LogrusLogger implements the Logger interface using logrus
type LogrusLogger struct {
	logger *logrus.Entry
}

var instance *LogrusLogger
var once sync.Once
var logFilePath string
var sharedLogger *logrus.Logger // Single shared logger instance

// CustomFormatter is our custom log formatter that ensures consistent formatting
type CustomFormatter struct {
	// Include level prefix in text format for easier frontend parsing
	IncludeLevelPrefix bool
}

// Format implements the logrus.Formatter interface
func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// Create a consistent log format with level prefix
	var levelPrefix string
	if f.IncludeLevelPrefix {
		switch entry.Level {
		case logrus.TraceLevel:
			levelPrefix = "[TRACE] "
		case logrus.DebugLevel:
			levelPrefix = "[DEBUG] "
		case logrus.InfoLevel:
			levelPrefix = "[INFO] "
		case logrus.WarnLevel:
			levelPrefix = "[WARN] "
		case logrus.ErrorLevel:
			levelPrefix = "[ERROR] "
		case logrus.FatalLevel:
			levelPrefix = "[FATAL] "
		case logrus.PanicLevel:
			levelPrefix = "[PANIC] "
		}
	}

	// Format the log with timestamp and level prefix
	timestamp := entry.Time.Format("2006/01/02 15:04:05")
	msg := entry.Message

	// Include any fields as key=value pairs
	fields := ""
	for k, v := range entry.Data {
		fields += " " + k + "=" + toString(v)
	}

	// Final log format: timestamp + level prefix + message + fields + newline
	log := []byte(timestamp + " " + levelPrefix + msg + fields + "\n")
	return log, nil
}

// toString converts a value to string for logging
func toString(value interface{}) string {
	if value == nil {
		return "nil"
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}

// GetLogger returns a singleton logger instance
func GetLogger() Logger {
	once.Do(func() {
		// If the shared logger hasn't been initialized yet, use the standard logger
		if sharedLogger == nil {
			sharedLogger = logrus.StandardLogger()
		}
		instance = &LogrusLogger{
			logger: logrus.NewEntry(sharedLogger),
		}
	})
	return instance
}

// Initialize sets up the logger with specified log level and file path
func Initialize(level, filePath string) error {
	// Initialize the shared logger instance
	sharedLogger = logrus.New()

	// Use our custom formatter with level prefixes
	customFormatter := &CustomFormatter{
		IncludeLevelPrefix: true,
	}
	sharedLogger.SetFormatter(customFormatter)

	// Parse the log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	sharedLogger.SetLevel(logLevel)

	// Create log directory if it doesn't exist
	if filePath != "" {
		logDir := filepath.Dir(filePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return err
		}

		// Open log file
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}

		// Write to both file and stdout
		writer := io.MultiWriter(os.Stdout, file)
		sharedLogger.SetOutput(writer)

		logFilePath = filePath
	} else {
		// Even if no file is specified, ensure logs go to stdout
		sharedLogger.SetOutput(os.Stdout)
	}

	// Also configure the standard logger to match our shared logger
	// This ensures any direct logrus calls use the same configuration
	logrus.SetFormatter(customFormatter)
	logrus.SetLevel(logLevel)
	if filePath != "" {
		logrus.SetOutput(sharedLogger.Out)
	} else {
		logrus.SetOutput(os.Stdout)
	}

	return nil
}

// Debug logs a debug message
func (l *LogrusLogger) Debug(args ...interface{}) {
	l.logger.Debug(args...)
}

// Debugf logs a formatted debug message
func (l *LogrusLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

// Info logs an info message
func (l *LogrusLogger) Info(args ...interface{}) {
	l.logger.Info(args...)
}

// Infof logs a formatted info message
func (l *LogrusLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

// Warn logs a warning message
func (l *LogrusLogger) Warn(args ...interface{}) {
	l.logger.Warn(args...)
}

// Warnf logs a formatted warning message
func (l *LogrusLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

// Error logs an error message
func (l *LogrusLogger) Error(args ...interface{}) {
	l.logger.Error(args...)
}

// Errorf logs a formatted error message
func (l *LogrusLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

// Fatal logs a fatal message
func (l *LogrusLogger) Fatal(args ...interface{}) {
	l.logger.Fatal(args...)
}

// Fatalf logs a formatted fatal message
func (l *LogrusLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
}

// Trace logs a trace message (highest verbosity)
func (l *LogrusLogger) Trace(args ...interface{}) {
	l.logger.Trace(args...)
}

// Tracef logs a formatted trace message
func (l *LogrusLogger) Tracef(format string, args ...interface{}) {
	l.logger.Tracef(format, args...)
}

// WithField returns a logger with a field added to its context
func (l *LogrusLogger) WithField(key string, value interface{}) Logger {
	return &LogrusLogger{
		logger: l.logger.WithField(key, value),
	}
}

// WithFields returns a logger with fields added to its context
func (l *LogrusLogger) WithFields(fields map[string]interface{}) Logger {
	return &LogrusLogger{
		logger: l.logger.WithFields(logrus.Fields(fields)),
	}
}

// GetLogFilePath returns the current log file path
func GetLogFilePath() string {
	return logFilePath
}

// UpdateLogLevel dynamically updates the log level for the shared logger
func UpdateLogLevel(level string) error {
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("invalid log level '%s': %w", level, err)
	}

	// Update the shared logger instance if it exists
	if sharedLogger != nil {
		sharedLogger.SetLevel(logLevel)
	}

	// Also update the standard logger for consistency
	logrus.SetLevel(logLevel)

	return nil
}
