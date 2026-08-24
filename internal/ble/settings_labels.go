package ble

import "fmt"

// SettingLabel resolves display names from the generated spec table.
// Unknown settings stay configurable with numeric labels: newer cameras
// may expose settings the table predates.
func SettingLabel(id byte) string {
	if def, ok := SettingDefs[id]; ok {
		return def.Name
	}
	return fmt.Sprintf("Setting %d", id)
}

// OptionLabel resolves an option value's display name for a setting.
func OptionLabel(id byte, value int64) string {
	if def, ok := SettingDefs[id]; ok {
		if name, ok := def.Options[value]; ok {
			return name
		}
	}
	return fmt.Sprintf("%d", value)
}
