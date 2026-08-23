package manager

import (
	"fmt"

	"github.com/dropz/dropz/internal/logging"
	"github.com/dropz/dropz/internal/model"
)

// GetConfig returns the current configuration
func (m *GoProManager) GetConfig() model.Config {
	return m.db.GetConfig()
}

// UpdateConfig updates the configuration
func (m *GoProManager) UpdateConfig(config model.Config) error {
	if err := m.db.UpdateConfig(config); err != nil {
		return err
	}
	m.log.Info("Config replaced", "sync_enabled", config.SyncEnabled,
		"pair_mode_enabled", config.PairModeEnabled, "log_level", config.LogLevel,
		"destination_folder", config.DestinationFolder)
	return nil
}

// setting describes one named config field: read, validate+write, and
// copy-from-defaults all derive from a single field accessor, so adding a
// setting is one table entry.
type setting struct {
	get  func(*model.Config) interface{}
	set  func(c *model.Config, name string, value interface{}) error
	copy func(dst, src *model.Config)
}

func boolSetting(field func(*model.Config) *bool) setting {
	return setting{
		get: func(c *model.Config) interface{} { return *field(c) },
		set: func(c *model.Config, name string, value interface{}) error {
			v, ok := value.(bool)
			if !ok {
				return fmt.Errorf("invalid value type for %s: expected bool", name)
			}
			*field(c) = v
			return nil
		},
		copy: func(dst, src *model.Config) { *field(dst) = *field(src) },
	}
}

// int32Setting validates min <= value <= max; max < 0 means no upper bound.
func int32Setting(field func(*model.Config) *int32, min, max int32, unit string) setting {
	return setting{
		get: func(c *model.Config) interface{} { return *field(c) },
		set: func(c *model.Config, name string, value interface{}) error {
			v, err := toInt32(value, name)
			if err != nil {
				return err
			}
			if max >= 0 && (v < min || v > max) {
				return fmt.Errorf("%s must be between %d and %d %s", name, min, max, unit)
			}
			if max < 0 && v < min {
				return fmt.Errorf("%s must be at least %d %s", name, min, unit)
			}
			*field(c) = v
			return nil
		},
		copy: func(dst, src *model.Config) { *field(dst) = *field(src) },
	}
}

func stringSetting(field func(*model.Config) *string, validate func(string) error) setting {
	return setting{
		get: func(c *model.Config) interface{} { return *field(c) },
		set: func(c *model.Config, name string, value interface{}) error {
			v, ok := value.(string)
			if !ok {
				return fmt.Errorf("invalid value type for %s: expected string", name)
			}
			if validate != nil {
				if err := validate(v); err != nil {
					return err
				}
			}
			*field(c) = v
			return nil
		},
		copy: func(dst, src *model.Config) { *field(dst) = *field(src) },
	}
}

func validateLogLevel(v string) error {
	if !logging.ValidLevel(v) {
		return fmt.Errorf("invalid log_level: expected one of trace, debug, info, warn, error, fatal")
	}
	return nil
}

var settings = map[string]setting{
	"pair_mode_enabled": boolSetting(func(c *model.Config) *bool { return &c.PairModeEnabled }),
	"sync_enabled":      boolSetting(func(c *model.Config) *bool { return &c.SyncEnabled }),
	"check_on_return":   boolSetting(func(c *model.Config) *bool { return &c.CheckOnReturn }),
	"set_time_enabled":  boolSetting(func(c *model.Config) *bool { return &c.SetTimeEnabled }),

	"scan_interval_seconds":         int32Setting(func(c *model.Config) *int32 { return &c.ScanIntervalSeconds }, 5, -1, "seconds"),
	"connect_timeout_seconds":       int32Setting(func(c *model.Config) *int32 { return &c.ConnectTimeoutSeconds }, 5, 300, "seconds"),
	"days_threshold":                int32Setting(func(c *model.Config) *int32 { return &c.DaysThreshold }, 1, -1, "day"),
	"inactivity_timeout_seconds":    int32Setting(func(c *model.Config) *int32 { return &c.InactivityTimeoutSeconds }, 30, 3600, "seconds"),
	"status_check_interval_seconds": int32Setting(func(c *model.Config) *int32 { return &c.StatusCheckIntervalSeconds }, 0, 3600, "seconds"),

	"destination_folder": stringSetting(func(c *model.Config) *string { return &c.DestinationFolder }, nil),
	"log_level":          stringSetting(func(c *model.Config) *string { return &c.LogLevel }, validateLogLevel),
}

// applyLogLevel applies the configured log level to the live logger.
// The change itself is logged by the caller (Setting updated / reset).
func (m *GoProManager) applyLogLevel(level string) {
	if m.logLevel == nil {
		return
	}
	parsed, err := logging.ParseLevel(level)
	if err != nil {
		m.log.Warn("Invalid log level not applied", "value", level, "err", err)
		return
	}
	m.logLevel.Set(parsed)
}

// GetSetting gets a specific setting value
func (m *GoProManager) GetSetting(settingName string) (interface{}, error) {
	def, ok := settings[settingName]
	if !ok {
		return nil, fmt.Errorf("unknown setting: %s", settingName)
	}
	config := m.db.GetConfig()
	return def.get(&config), nil
}

// UpdateSetting updates a specific setting with validation
func (m *GoProManager) UpdateSetting(settingName string, value interface{}) (model.Config, error) {
	config := m.db.GetConfig()

	def, ok := settings[settingName]
	if !ok {
		return config, fmt.Errorf("unknown setting: %s", settingName)
	}
	old := def.get(&config)
	if err := def.set(&config, settingName, value); err != nil {
		return config, err
	}
	if settingName == "log_level" {
		m.applyLogLevel(config.LogLevel)
	}

	if err := m.db.UpdateConfig(config); err != nil {
		return config, fmt.Errorf("failed to update config: %w", err)
	}
	m.log.Info("Setting updated", "setting", settingName, "old", old, "new", def.get(&config))
	return config, nil
}

// ResetSetting resets a specific setting (or "all") to its default value
func (m *GoProManager) ResetSetting(settingName string) (model.Config, error) {
	config := m.db.GetConfig()
	defaults := model.DefaultConfig()

	if settingName == "all" {
		config = defaults
		m.applyLogLevel(config.LogLevel)
	} else {
		def, ok := settings[settingName]
		if !ok {
			return config, fmt.Errorf("unknown setting: %s", settingName)
		}
		def.copy(&config, &defaults)
		if settingName == "log_level" {
			m.applyLogLevel(config.LogLevel)
		}
	}

	if err := m.db.UpdateConfig(config); err != nil {
		return config, fmt.Errorf("failed to update config: %w", err)
	}
	m.log.Info("Setting reset to default", "setting", settingName)
	return config, nil
}

func toInt32(value interface{}, name string) (int32, error) {
	switch v := value.(type) {
	case float64:
		return int32(v), nil
	case int:
		return int32(v), nil
	case int32:
		return v, nil
	default:
		return 0, fmt.Errorf("invalid value type for %s: expected number", name)
	}
}
