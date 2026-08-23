package manager

import (
	"testing"
)

func TestUpdateSettingValidation(t *testing.T) {
	m := newTestManager(t)

	cases := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"pair_mode_enabled", true, false},
		{"pair_mode_enabled", "yes", true},
		{"scan_interval_seconds", int32(30), false},
		{"scan_interval_seconds", int32(2), true}, // below min 5
		{"connect_timeout_seconds", int32(301), true},
		{"connect_timeout_seconds", int32(120), false},
		{"days_threshold", int32(0), true},
		{"inactivity_timeout_seconds", int32(29), true},
		{"status_check_interval_seconds", int32(0), false}, // 0 = disabled
		{"status_check_interval_seconds", int32(3601), true},
		{"log_level", "debug", false},
		{"log_level", "loud", true},
		{"destination_folder", "/tmp/x", false},
		{"no_such_setting", 1, true},
	}

	for _, c := range cases {
		_, err := m.UpdateSetting(c.name, c.value)
		if (err != nil) != c.wantErr {
			t.Errorf("UpdateSetting(%s, %v) err = %v, wantErr %v", c.name, c.value, err, c.wantErr)
		}
	}
}

func TestUpdateSettingPersists(t *testing.T) {
	m := newTestManager(t)

	if _, err := m.UpdateSetting("days_threshold", int32(14)); err != nil {
		t.Fatal(err)
	}
	if got := m.GetConfig().DaysThreshold; got != 14 {
		t.Errorf("DaysThreshold = %d, want 14", got)
	}

	v, err := m.GetSetting("days_threshold")
	if err != nil || v.(int32) != 14 {
		t.Errorf("GetSetting = %v, %v", v, err)
	}
}

func TestResetSetting(t *testing.T) {
	m := newTestManager(t)
	defaults := m.GetConfig()

	m.UpdateSetting("days_threshold", int32(14))
	cfg, err := m.ResetSetting("days_threshold")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DaysThreshold != defaults.DaysThreshold {
		t.Errorf("reset DaysThreshold = %d, want %d", cfg.DaysThreshold, defaults.DaysThreshold)
	}

	if _, err := m.ResetSetting("bogus"); err == nil {
		t.Error("ResetSetting must reject unknown names")
	}

	m.UpdateSetting("days_threshold", int32(14))
	m.UpdateSetting("sync_enabled", false)
	cfg, err = m.ResetSetting("all")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DaysThreshold != defaults.DaysThreshold || cfg.SyncEnabled != defaults.SyncEnabled {
		t.Errorf("reset all incomplete: %+v", cfg)
	}
}

func TestGetSettingUnknown(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.GetSetting("bogus"); err == nil {
		t.Error("GetSetting must reject unknown names")
	}
}
