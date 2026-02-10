package manager

import (
	"fmt"
	"strings"

	"github.com/dropz/dropz/pkg/database"
	"github.com/sirupsen/logrus"
)

// GetConfig returns the current configuration
func (m *GoProManager) GetConfig() database.Config {
	return m.db.GetConfig()
}

// UpdateConfig updates the configuration
func (m *GoProManager) UpdateConfig(config database.Config) error {
	return m.db.UpdateConfig(config)
}

// GetSetting gets a specific setting value
func (m *GoProManager) GetSetting(settingName string) (interface{}, error) {
	config := m.db.GetConfig()

	switch settingName {
	case "sync_enabled":
		return config.SyncEnabled, nil
	case "scan_interval_seconds":
		return config.ScanIntervalSeconds, nil
	case "connect_timeout_seconds":
		return config.ConnectTimeoutSeconds, nil
	case "days_threshold":
		return config.DaysThreshold, nil
	case "destination_folder":
		return config.DestinationFolder, nil
	case "inactivity_timeout_seconds":
		return config.InactivityTimeoutSeconds, nil
	case "inactivity_sync_interval_seconds":
		return config.InactivitySyncIntervalSeconds, nil
	case "set_time_enabled":
		return config.SetTimeEnabled, nil
	case "log_level":
		return config.LogLevel, nil
	default:
		return nil, fmt.Errorf("unknown setting: %s", settingName)
	}
}

// UpdateSetting updates a specific setting with validation
func (m *GoProManager) UpdateSetting(settingName string, value interface{}) (database.Config, error) {
	config := m.db.GetConfig()

	switch settingName {
	case "pair_mode_enabled":
		boolVal, ok := value.(bool)
		if !ok {
			return config, fmt.Errorf("invalid value type for pair_mode_enabled: expected bool")
		}
		config.PairModeEnabled = boolVal

	case "sync_enabled":
		boolVal, ok := value.(bool)
		if !ok {
			return config, fmt.Errorf("invalid value type for sync_enabled: expected bool")
		}
		config.SyncEnabled = boolVal

	case "scan_interval_seconds":
		intVal, err := toInt32(value, "scan_interval_seconds")
		if err != nil {
			return config, err
		}
		if intVal < 5 {
			return config, fmt.Errorf("scan_interval_seconds must be at least 5 seconds")
		}
		config.ScanIntervalSeconds = intVal

	case "connect_timeout_seconds":
		intVal, err := toInt32(value, "connect_timeout_seconds")
		if err != nil {
			return config, err
		}
		if intVal < 5 || intVal > 60 {
			return config, fmt.Errorf("connect_timeout_seconds must be between 5 and 60 seconds")
		}
		config.ConnectTimeoutSeconds = intVal

	case "days_threshold":
		intVal, err := toInt32(value, "days_threshold")
		if err != nil {
			return config, err
		}
		if intVal < 1 {
			return config, fmt.Errorf("days_threshold must be at least 1 day")
		}
		config.DaysThreshold = intVal

	case "destination_folder":
		strVal, ok := value.(string)
		if !ok {
			return config, fmt.Errorf("invalid value type for destination_folder: expected string")
		}
		config.DestinationFolder = strVal

	case "inactivity_timeout_seconds":
		intVal, err := toInt32(value, "inactivity_timeout_seconds")
		if err != nil {
			return config, err
		}
		if intVal < 30 || intVal > 3600 {
			return config, fmt.Errorf("inactivity_timeout_seconds must be between 30 and 3600 seconds")
		}
		config.InactivityTimeoutSeconds = intVal

	case "inactivity_sync_interval_seconds":
		intVal, err := toInt32(value, "inactivity_sync_interval_seconds")
		if err != nil {
			return config, err
		}
		if intVal < 60 || intVal > 3600 {
			return config, fmt.Errorf("inactivity_sync_interval_seconds must be between 60 and 3600 seconds")
		}
		config.InactivitySyncIntervalSeconds = intVal

	case "set_time_enabled":
		boolVal, ok := value.(bool)
		if !ok {
			return config, fmt.Errorf("invalid value type for set_time_enabled: expected bool")
		}
		config.SetTimeEnabled = boolVal

	case "log_level":
		strVal, ok := value.(string)
		if !ok {
			return config, fmt.Errorf("invalid value type for log_level: expected string")
		}
		validLevels := map[string]bool{
			"trace": true, "debug": true, "info": true,
			"warn": true, "error": true, "fatal": true,
		}
		if !validLevels[strings.ToLower(strVal)] {
			return config, fmt.Errorf("invalid log_level: expected one of trace, debug, info, warn, error, fatal")
		}
		config.LogLevel = strVal
		if level, err := logrus.ParseLevel(strVal); err == nil {
			m.log.SetLevel(level)
		}

	default:
		return config, fmt.Errorf("unknown setting: %s", settingName)
	}

	if err := m.db.UpdateConfig(config); err != nil {
		return config, fmt.Errorf("failed to update config: %w", err)
	}

	return config, nil
}

// ResetSetting resets a specific setting to its default value
func (m *GoProManager) ResetSetting(settingName string) (database.Config, error) {
	config := m.db.GetConfig()
	defaults := database.DefaultConfig()

	switch settingName {
	case "pair_mode_enabled":
		config.PairModeEnabled = defaults.PairModeEnabled
	case "sync_enabled":
		config.SyncEnabled = defaults.SyncEnabled
	case "scan_interval_seconds":
		config.ScanIntervalSeconds = defaults.ScanIntervalSeconds
	case "connect_timeout_seconds":
		config.ConnectTimeoutSeconds = defaults.ConnectTimeoutSeconds
	case "days_threshold":
		config.DaysThreshold = defaults.DaysThreshold
	case "destination_folder":
		config.DestinationFolder = defaults.DestinationFolder
	case "inactivity_timeout_seconds":
		config.InactivityTimeoutSeconds = defaults.InactivityTimeoutSeconds
	case "inactivity_sync_interval_seconds":
		config.InactivitySyncIntervalSeconds = defaults.InactivitySyncIntervalSeconds
	case "set_time_enabled":
		config.SetTimeEnabled = defaults.SetTimeEnabled
	case "log_level":
		config.LogLevel = defaults.LogLevel
		if level, err := logrus.ParseLevel(defaults.LogLevel); err == nil {
			m.log.SetLevel(level)
		}
	case "all":
		config = defaults
		if level, err := logrus.ParseLevel(defaults.LogLevel); err == nil {
			m.log.SetLevel(level)
		}
	default:
		return config, fmt.Errorf("unknown setting: %s", settingName)
	}

	if err := m.db.UpdateConfig(config); err != nil {
		return config, fmt.Errorf("failed to update config: %w", err)
	}

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
