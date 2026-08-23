package manager

import (
	"testing"
)

func TestBuildSettingsSnapshot(t *testing.T) {
	values := map[byte]int64{2: 9, 3: 0, 250: 7}
	caps := map[byte][]int64{2: {12, 9, 1}, 3: {0}}

	settings := buildSettingsSnapshot(values, caps)

	if len(settings) != 3 {
		t.Fatalf("got %d settings, want 3", len(settings))
	}
	if settings[0].ID != 2 || settings[1].ID != 3 || settings[2].ID != 250 {
		t.Errorf("not sorted by ID: %+v", settings)
	}

	res := settings[0]
	if res.Name != "Video Resolution" || res.ValueName != "1080" {
		t.Errorf("labels wrong: %+v", res)
	}
	if len(res.Options) != 3 || res.Options[0].Value != 1 || res.Options[2].Value != 12 {
		t.Errorf("options not sorted: %+v", res.Options)
	}
	if res.Options[2].Name != "720" {
		t.Errorf("option label = %q, want 720", res.Options[2].Name)
	}

	// Unknown setting stays configurable with numeric labels.
	unknown := settings[2]
	if unknown.Name != "Setting 250" || unknown.ValueName != "7" {
		t.Errorf("unknown setting labels: %+v", unknown)
	}
	if len(unknown.Options) != 0 {
		t.Errorf("unknown setting should have no options: %+v", unknown)
	}
}
