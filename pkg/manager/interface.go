package manager

import (
	"github.com/dropz/dropz/pkg/database"
)

// Configuration delegation methods
func (m *GoProManager) GetConfig() database.Config {
	return m.configManager.GetConfig()
}

func (m *GoProManager) UpdateConfig(config database.Config) error {
	return m.configManager.UpdateConfig(config)
}

func (m *GoProManager) GetSetting(settingName string) (interface{}, error) {
	return m.configManager.GetSetting(settingName)
}

func (m *GoProManager) UpdateSetting(settingName string, value interface{}) (database.Config, error) {
	return m.configManager.UpdateSetting(settingName, value)
}

func (m *GoProManager) ResetSetting(settingName string) (database.Config, error) {
	return m.configManager.ResetSetting(settingName)
}

// Group delegation methods
func (m *GoProManager) CreateGroup(name string, cameraIDs []string) (*database.Group, error) {
	return m.groupManager.CreateGroup(name, cameraIDs)
}

func (m *GoProManager) UpdateGroup(groupID, name string, cameraIDs []string) (*database.Group, error) {
	return m.groupManager.UpdateGroup(groupID, name, cameraIDs)
}

func (m *GoProManager) DeleteGroup(groupID string) error {
	return m.groupManager.DeleteGroup(groupID)
}

// Video delegation methods  
func (m *GoProManager) GetAllVideos() []*database.VideoFile {
	// For now, return empty slice - this method needs to be implemented in sync coordinator
	return []*database.VideoFile{}
}
