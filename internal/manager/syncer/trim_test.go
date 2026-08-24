package syncer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapToKeyframe(t *testing.T) {
	keys := []int64{0, 1001, 2002, 3003}
	cases := map[int64]int64{0: 0, 500: 0, 1001: 1001, 1500: 1001, 9999: 3003}
	for in, want := range cases {
		if got := snapToKeyframe(keys, in); got != want {
			t.Errorf("snap(%d) = %d, want %d", in, got, want)
		}
	}
	if got := snapToKeyframe(nil, 1234); got != 1234 {
		t.Errorf("no sync table: got %d, want the input", got)
	}
}

func TestTrimNameAvoidsCollisions(t *testing.T) {
	dir := t.TempDir()
	if got := trimName(dir, "GX010064.MP4", 11011, 45000); got != "GX010064_trim-0011-0045.MP4" {
		t.Errorf("name = %q", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "GX010064_trim-0011-0045.MP4"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := trimName(dir, "GX010064.MP4", 11011, 45000); got != "GX010064_trim-0011-0045-2.MP4" {
		t.Errorf("second name = %q", got)
	}
	if got := msArg(11011); got != "11.011" {
		t.Errorf("msArg = %q", got)
	}
}
