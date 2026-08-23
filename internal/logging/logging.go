// Package logging builds the process-wide slog logger.
// One writer per line: stderr and the rotating file share one handler,
// so a line is never formatted twice or written by two owners.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// LevelTrace sits below slog.LevelDebug; enabled by log_level=trace.
const LevelTrace = slog.LevelDebug - 4

// levelNames maps config vocabulary to slog levels.
// fatal maps to error: slog has no fatal, callers os.Exit themselves.
var levelNames = map[string]slog.Level{
	"trace": LevelTrace,
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
	"fatal": slog.LevelError,
}

// ParseLevel converts a config level name to a slog.Level.
func ParseLevel(name string) (slog.Level, error) {
	if lvl, ok := levelNames[strings.ToLower(name)]; ok {
		return lvl, nil
	}
	return 0, fmt.Errorf("invalid log level %q: expected one of trace, debug, info, warn, error, fatal", name)
}

// ValidLevel reports whether name is an accepted log level.
func ValidLevel(name string) bool {
	_, ok := levelNames[strings.ToLower(name)]
	return ok
}

// Options configures New.
type Options struct {
	Level    string // trace|debug|info|warn|error|fatal
	Format   string // "text" (default) or "json"; json is for the Electron host
	FilePath string // rotating log file; empty = stderr only
}

// replaceTrace renders the custom trace level by name instead of "DEBUG-4".
func replaceTrace(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		if lvl, ok := a.Value.Any().(slog.Level); ok && lvl == LevelTrace {
			a.Value = slog.StringValue("TRACE")
		}
	}
	return a
}

// New builds the root logger. The returned LevelVar changes verbosity at
// runtime (settings RPC) without rebuilding handlers.
func New(opts Options) (*slog.Logger, *slog.LevelVar, error) {
	levelVar := new(slog.LevelVar)
	if opts.Level != "" {
		lvl, err := ParseLevel(opts.Level)
		if err != nil {
			return nil, nil, err
		}
		levelVar.Set(lvl)
	}

	var out io.Writer = os.Stderr
	if opts.FilePath != "" {
		out = io.MultiWriter(os.Stderr, &lumberjack.Logger{
			Filename:   opts.FilePath,
			MaxSize:    5, // MB
			MaxBackups: 3,
			Compress:   true,
		})
	}

	hopts := &slog.HandlerOptions{Level: levelVar, ReplaceAttr: replaceTrace}
	var handler slog.Handler
	switch strings.ToLower(opts.Format) {
	case "", "text":
		handler = slog.NewTextHandler(out, hopts)
	case "json":
		handler = slog.NewJSONHandler(out, hopts)
	default:
		return nil, nil, fmt.Errorf("invalid log format %q: expected text or json", opts.Format)
	}

	return slog.New(handler), levelVar, nil
}

// Trace logs at LevelTrace; slog has no built-in helper for custom levels.
func Trace(l *slog.Logger, msg string, args ...any) {
	l.Log(context.Background(), LevelTrace, msg, args...)
}
