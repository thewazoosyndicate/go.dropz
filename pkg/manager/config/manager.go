package config

import (
	"fmt"
	"time"

	"github.com/dropz/dropz/pkg/database"
	"github.com/sirupsen/logrus"
)

// Manager handles configuration operations
type Manager struct {
	db  *database.Database
	log *logrus.Logger
	
	// Configuration state
	scanInterval     time.Duration
	connectTimeout   time.Duration
	inactivityTime   time.Duration
	setTimeEnabled   bool
	daysThreshold    int
}

// NewManager creates a new configuration manager
func NewManager(db *database.Database, log *logrus.Logger) *Manager {
	config := db.GetConfig()
	
	return &Manager{
		db:               db,
		log:              log,
		scanInterval:     time.Duration(config.ScanIntervalSeconds) * time.Second,
		connectTimeout:   time.Duration(config.ConnectTimeoutSeconds) * time.Second,
		inactivityTime:   time.Duration(config.InactivityTimeoutSeconds) * time.Second,
		setTimeEnabled:   config.SetTimeEnabled,
		daysThreshold:    int(config.DaysThreshold),
	}
}

// GetConfig returns the current configuration
func (cm *Manager) GetConfig() database.Config {
	return cm.db.GetConfig()
}

// UpdateConfig updates the configuration and internal state
func (cm *Manager) UpdateConfig(config database.Config) error {
	// Update local variables
	cm.scanInterval = time.Duration(config.ScanIntervalSeconds) * time.Second
	cm.connectTimeout = time.Duration(config.ConnectTimeoutSeconds) * time.Second
	cm.inactivityTime = time.Duration(config.InactivityTimeoutSeconds) * time.Second
	cm.setTimeEnabled = config.SetTimeEnabled
	cm.daysThreshold = int(config.DaysThreshold)

	// Update database
	return cm.db.UpdateConfig(config)
}

// GetSetting gets a specific setting value
func (cm *Manager) GetSetting(settingName string) (interface{}, error) {
	config := cm.db.GetConfig()

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

// GetScanInterval returns the current scan interval
func (cm *Manager) GetScanInterval() time.Duration {
	return cm.scanInterval
}

// GetConnectTimeout returns the current connect timeout
func (cm *Manager) GetConnectTimeout() time.Duration {
	return cm.connectTimeout
}

// GetInactivityTime returns the current inactivity timeout
func (cm *Manager) GetInactivityTime() time.Duration {
	return cm.inactivityTime
}

// IsSetTimeEnabled returns whether time setting is enabled
func (cm *Manager) IsSetTimeEnabled() bool {
	return cm.setTimeEnabled
}

// GetDaysThreshold returns the current days threshold
func (cm *Manager) GetDaysThreshold() int {
	return cm.daysThreshold
}
