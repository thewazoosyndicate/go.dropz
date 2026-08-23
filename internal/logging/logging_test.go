package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"trace": LevelTrace,
		"debug": slog.LevelDebug,
		"INFO":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
		"fatal": slog.LevelError,
	}
	for name, want := range cases {
		got, err := ParseLevel(name)
		if err != nil || got != want {
			t.Errorf("ParseLevel(%q) = %v, %v; want %v", name, got, err, want)
		}
	}
	if _, err := ParseLevel("loud"); err == nil {
		t.Error("ParseLevel must reject unknown names")
	}
}

// TestJSONContract pins the line shape the Electron host parses:
// top-level time, level, msg, plus flat attr keys.
func TestJSONContract(t *testing.T) {
	var buf bytes.Buffer
	lv := new(slog.LevelVar)
	lv.Set(LevelTrace)
	log := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: lv, ReplaceAttr: replaceTrace}))
	log = log.With("component", "ble")

	log.Warn("Connection attempt failed", "attempt", 2, "err", errors.New("boom"))

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("line is not valid JSON: %v (%s)", err, buf.String())
	}
	for _, key := range []string{"time", "level", "msg", "component", "attempt", "err"} {
		if _, ok := line[key]; !ok {
			t.Errorf("missing key %q in %s", key, buf.String())
		}
	}
	if line["level"] != "WARN" {
		t.Errorf("level = %v, want WARN", line["level"])
	}

	// Trace renders by name, not DEBUG-4
	buf.Reset()
	Trace(log, "probe")
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatal(err)
	}
	if line["level"] != "TRACE" {
		t.Errorf("trace level = %v, want TRACE", line["level"])
	}
}
