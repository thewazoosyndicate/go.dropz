package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/dropz/dropz/pkg/database"
)

// UpdateSetting updates a specific setting with validation
func (cm *Manager) UpdateSetting(settingName string, value interface{}) (database.Config, error) {
	config := cm.db.GetConfig()

	switch settingName {
	case "pair_mode_enabled":
		boolVal, ok := value.(bool)
		if !ok {
			return config, fmt.Errorf("invalid value type for pair_mode_enabled: expected bool")
		}
		config.PairModeEnabled = boolVal
		cm.log.Infof("Updated pair_mode_enabled to %v", boolVal)

	case "sync_enabled":
		boolVal, ok := value.(bool)
		if !ok {
			return config, fmt.Errorf("invalid value type for sync_enabled: expected bool")
		}
		config.SyncEnabled = boolVal
		cm.log.Infof("Updated sync_enabled to %v", boolVal)

	case "scan_interval_seconds":
		var intVal int32
		switch v := value.(type) {
		case float64:
			intVal = int32(v)
		case int:
			intVal = int32(v)
		case int32:
			intVal = v
		default:
			return config, fmt.Errorf("invalid value type for scan_interval_seconds: expected number")
		}
		if intVal < 5 {
			return config, fmt.Errorf("scan_interval_seconds must be at least 5 seconds")
		}
		config.ScanIntervalSeconds = intVal
		cm.log.Infof("Updated scan_interval_seconds to %d", intVal)

	case "connect_timeout_seconds":
		var intVal int32
		switch v := value.(type) {
		case float64:
			intVal = int32(v)
		case int:
			intVal = int32(v)
		case int32:
			intVal = v
		default:
			return config, fmt.Errorf("invalid value type for connect_timeout_seconds: expected number")
		}
		if intVal < 5 || intVal > 60 {
			return config, fmt.Errorf("connect_timeout_seconds must be between 5 and 60 seconds")
		}
		config.ConnectTimeoutSeconds = intVal
		cm.log.Infof("Updated connect_timeout_seconds to %d", intVal)

	case "days_threshold":
		var intVal int32
		switch v := value.(type) {
		case float64:
			intVal = int32(v)
		case int:
			intVal = int32(v)
		case int32:
			intVal = v
		default:
			return config, fmt.Errorf("invalid value type for days_threshold: expected number")
		}
		if intVal < 1 {
			return config, fmt.Errorf("days_threshold must be at least 1 day")
		}
		config.DaysThreshold = intVal
		cm.log.Infof("Updated days_threshold to %d", intVal)

	case "destination_folder":
		strVal, ok := value.(string)
		if !ok {
			return config, fmt.Errorf("invalid value type for destination_folder: expected string")
		}
		config.DestinationFolder = strVal
		cm.log.Infof("Updated destination_folder to %s", strVal)

	case "inactivity_timeout_seconds":
		var intVal int32
		switch v := value.(type) {
		case float64:
			intVal = int32(v)
		case int:
			intVal = int32(v)
		case int32:
			intVal = v
		default:
			return config, fmt.Errorf("invalid value type for inactivity_timeout_seconds: expected number")
		}
		if intVal < 30 || intVal > 3600 {
			return config, fmt.Errorf("inactivity_timeout_seconds must be between 30 and 3600 seconds")
		}
		config.InactivityTimeoutSeconds = intVal
		cm.log.Infof("Updated inactivity_timeout_seconds to %d", intVal)

	case "inactivity_sync_interval_seconds":
		var intVal int32
		switch v := value.(type) {
		case float64:
			intVal = int32(v)
		case int:
			intVal = int32(v)
		case int32:
			intVal = v
		default:
			return config, fmt.Errorf("invalid value type for inactivity_sync_interval_seconds: expected number")
		}
		if intVal < 60 || intVal > 3600 {
			return config, fmt.Errorf("inactivity_sync_interval_seconds must be between 60 and 3600 seconds")
		}
		config.InactivitySyncIntervalSeconds = intVal
		cm.log.Infof("Updated inactivity_sync_interval_seconds to %d", intVal)

	case "set_time_enabled":
		boolVal, ok := value.(bool)
		if !ok {
			return config, fmt.Errorf("invalid value type for set_time_enabled: expected bool")
		}
		config.SetTimeEnabled = boolVal
		cm.log.Infof("Updated set_time_enabled to %v", boolVal)

	case "log_level":
		strVal, ok := value.(string)
		if !ok {
			return config, fmt.Errorf("invalid value type for log_level: expected string")
		}
		validLogLevels := map[string]bool{
			"trace": true,
			"debug": true,
			"info":  true,
			"warn":  true,
			"error": true,
			"fatal": true,
		}
		if !validLogLevels[strings.ToLower(strVal)] {
			return config, fmt.Errorf("invalid log_level: expected one of trace, debug, info, warn, error, fatal")
		}
		config.LogLevel = strVal
		cm.log.Infof("Updated log_level to %s", strVal)

	default:
		return config, fmt.Errorf("unknown setting: %s", settingName)
	}

	// Update the database
	if err := cm.db.UpdateConfig(config); err != nil {
		return config, fmt.Errorf("failed to update config in database: %w", err)
	}

	// Update local variables
	cm.scanInterval = time.Duration(config.ScanIntervalSeconds) * time.Second
	cm.connectTimeout = time.Duration(config.ConnectTimeoutSeconds) * time.Second
	cm.inactivityTime = time.Duration(config.InactivityTimeoutSeconds) * time.Second
	cm.setTimeEnabled = config.SetTimeEnabled
	cm.daysThreshold = int(config.DaysThreshold)

	return config, nil
}

// ResetSetting resets a specific setting to its default value
func (cm *Manager) ResetSetting(settingName string) (database.Config, error) {
	config := cm.db.GetConfig()
	defaultConfig := database.DefaultConfig()

	switch settingName {
	case "pair_mode_enabled":
		config.PairModeEnabled = defaultConfig.PairModeEnabled
		cm.log.Infof("Reset pair_mode_enabled to default: %v", defaultConfig.PairModeEnabled)

	case "sync_enabled":
		config.SyncEnabled = defaultConfig.SyncEnabled
		cm.log.Infof("Reset sync_enabled to default: %v", defaultConfig.SyncEnabled)

	case "scan_interval_seconds":
		config.ScanIntervalSeconds = defaultConfig.ScanIntervalSeconds
		cm.log.Infof("Reset scan_interval_seconds to default: %d", defaultConfig.ScanIntervalSeconds)

	case "connect_timeout_seconds":
		config.ConnectTimeoutSeconds = defaultConfig.ConnectTimeoutSeconds
		cm.log.Infof("Reset connect_timeout_seconds to default: %d", defaultConfig.ConnectTimeoutSeconds)

	case "days_threshold":
		config.DaysThreshold = defaultConfig.DaysThreshold
		cm.log.Infof("Reset days_threshold to default: %d", defaultConfig.DaysThreshold)

	case "destination_folder":
		config.DestinationFolder = defaultConfig.DestinationFolder
		cm.log.Infof("Reset destination_folder to default: %s", defaultConfig.DestinationFolder)

	case "inactivity_timeout_seconds":
		config.InactivityTimeoutSeconds = defaultConfig.InactivityTimeoutSeconds
		cm.log.Infof("Reset inactivity_timeout_seconds to default: %d", defaultConfig.InactivityTimeoutSeconds)

	case "inactivity_sync_interval_seconds":
		config.InactivitySyncIntervalSeconds = defaultConfig.InactivitySyncIntervalSeconds
		cm.log.Infof("Reset inactivity_sync_interval_seconds to default: %d", defaultConfig.InactivitySyncIntervalSeconds)

	case "set_time_enabled":
		config.SetTimeEnabled = defaultConfig.SetTimeEnabled
		cm.log.Infof("Reset set_time_enabled to default: %v", defaultConfig.SetTimeEnabled)

	case "log_level":
		config.LogLevel = defaultConfig.LogLevel
		cm.log.Infof("Reset log_level to default: %s", defaultConfig.LogLevel)

	case "all":
		// Reset all settings to default
		config = defaultConfig
		cm.log.Info("Reset all settings to default values")

	default:
		return config, fmt.Errorf("unknown setting: %s", settingName)
	}

	// Update the database
	if err := cm.db.UpdateConfig(config); err != nil {
		return config, fmt.Errorf("failed to update config in database: %w", err)
	}

	// Update local variables
	cm.scanInterval = time.Duration(config.ScanIntervalSeconds) * time.Second
	cm.connectTimeout = time.Duration(config.ConnectTimeoutSeconds) * time.Second
	cm.inactivityTime = time.Duration(config.InactivityTimeoutSeconds) * time.Second
	cm.setTimeEnabled = config.SetTimeEnabled
	cm.daysThreshold = int(config.DaysThreshold)

	return config, nil
}
