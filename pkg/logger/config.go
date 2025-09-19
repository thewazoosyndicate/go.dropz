package logger

import (
	"io"

	"github.com/sirupsen/logrus"
)

// Log levels
const (
	LevelTrace = "trace"
	LevelDebug = "debug" // More verbose than trace
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
	LevelFatal = "fatal"
)

// SetOutput configures output for specific log level
func SetOutput(level string, writer io.Writer) error {
	// Get the logrus level
	var logrusLevel logrus.Level
	switch level {
	case LevelTrace:
		logrusLevel = logrus.TraceLevel
	case LevelDebug:
		logrusLevel = logrus.DebugLevel
	case LevelInfo:
		logrusLevel = logrus.InfoLevel
	case LevelWarn:
		logrusLevel = logrus.WarnLevel
	case LevelError:
		logrusLevel = logrus.ErrorLevel
	case LevelFatal:
		logrusLevel = logrus.FatalLevel
	default:
		return nil // Ignore unknown levels
	}

	log := GetLogger()
	if log != nil {
		// Configure logrus hooks to route specific levels to specific writers
		log.Hooks.Add(&levelOutputHook{
			level:  logrusLevel,
			writer: writer,
		})
	}

	return nil
}

// levelOutputHook is a hook to route specific levels to specific writers
type levelOutputHook struct {
	level  logrus.Level
	writer io.Writer
}

// Levels returns the levels this hook should be applied to
func (h *levelOutputHook) Levels() []logrus.Level {
	return []logrus.Level{h.level}
}

// Fire executes the hook action
func (h *levelOutputHook) Fire(entry *logrus.Entry) error {
	// Format the entry
	line, err := entry.Logger.Formatter.Format(entry)
	if err != nil {
		return err
	}

	// Write to the specific writer for this level
	_, err = h.writer.Write(line)
	return err
}
