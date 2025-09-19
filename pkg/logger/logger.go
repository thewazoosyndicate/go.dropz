package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	instance *logrus.Logger
	once sync.Once
	logFilePath string
	initMutex sync.Mutex
	initialized bool
)

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
func GetLogger() *logrus.Logger {
	once.Do(func() {
		initMutex.Lock()
		defer initMutex.Unlock()
		
		if !initialized {
			// If Initialize hasn't been called, create a default logger
			instance = logrus.New()
			instance.SetLevel(logrus.InfoLevel)
			instance.SetOutput(os.Stdout)
			instance.SetFormatter(&CustomFormatter{
				IncludeLevelPrefix: true,
			})
			initialized = true
		}
	})
	return instance
}

// Initialize sets up the logger with specified log level and file path
func Initialize(level, filePath string) error {
	initMutex.Lock()
	defer initMutex.Unlock()
	
	// If already initialized through GetLogger, reuse the instance
	if instance == nil {
		instance = logrus.New()
	}

	// Use our custom formatter with level prefixes
	customFormatter := &CustomFormatter{
		IncludeLevelPrefix: true,
	}
	instance.SetFormatter(customFormatter)

	// Parse the log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	instance.SetLevel(logLevel)

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
		instance.SetOutput(writer)

		logFilePath = filePath
	} else {
		// Even if no file is specified, ensure logs go to stdout
		instance.SetOutput(os.Stdout)
	}

	// Also configure the standard logger to match our instance
	// This ensures any direct logrus calls use the same configuration
	logrus.SetFormatter(customFormatter)
	logrus.SetLevel(logLevel)
	if filePath != "" {
		logrus.SetOutput(instance.Out)
	} else {
		logrus.SetOutput(os.Stdout)
	}
	
	// Mark as initialized
	initialized = true

	return nil
}


// GetLogFilePath returns the current log file path
func GetLogFilePath() string {
	return logFilePath
}

// UpdateLogLevel dynamically updates the log level
func UpdateLogLevel(level string) error {
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("invalid log level '%s': %w", level, err)
	}

	// Update the logger instance if it exists
	if instance != nil {
		instance.SetLevel(logLevel)
	}

	// Also update the standard logger for consistency
	logrus.SetLevel(logLevel)

	return nil
}
