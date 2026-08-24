package wifi

import (
	"reflect"
	"testing"
)

func TestLRVCameraPath(t *testing.T) {
	cases := map[string]struct {
		want string
		ok   bool
	}{
		"100GOPRO/GX019795.MP4": {"100GOPRO/GL019795.LRV", true},
		"101GOPRO/GH010002.MP4": {"101GOPRO/GL010002.LRV", true},
		"100GOPRO/G0010001.JPG": {"", false},
		"100GOPRO/GL019795.LRV": {"", false},
	}
	for in, c := range cases {
		got, ok := LRVCameraPath(in)
		if got != c.want || ok != c.ok {
			t.Errorf("LRVCameraPath(%q) = %q,%v want %q,%v", in, got, ok, c.want, c.ok)
		}
	}
}

// HERO13 cards reuse names across GOPRO directories after folder rollover;
// duplicated names get the directory prefix, unique ones stay bare.
func TestLocalMediaNameDisambiguatesDuplicates(t *testing.T) {
	files := []MediaFile{
		{Name: "GX010002.MP4", CameraPath: "101GOPRO/GX010002.MP4"},
		{Name: "GX010002.MP4", CameraPath: "102GOPRO/GX010002.MP4"},
		{Name: "GX010003.MP4", CameraPath: "101GOPRO/GX010003.MP4"},
	}
	dupes := DuplicateNames(files)
	var got []string
	for _, f := range files {
		got = append(got, LocalMediaName(f.CameraPath, f.Name, dupes[f.Name]))
	}
	want := []string{"101GOPRO_GX010002.MP4", "102GOPRO_GX010002.MP4", "GX010003.MP4"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("local names = %v, want %v", got, want)
	}
}

func TestExpandGroupMembers(t *testing.T) {
	got := expandGroupMembers("G0010001.JPG", "1", "5", []string{"3"})
	want := []string{"G0010002.JPG", "G0010004.JPG", "G0010005.JPG"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("members = %v, want %v", got, want)
	}

	if got := expandGroupMembers("GOPR0001.JPG", "x", "5", nil); got != nil {
		t.Errorf("non-numeric first id must yield nil, got %v", got)
	}
	if got := expandGroupMembers("SHORT.JPG", "1", "5", nil); got != nil {
		t.Errorf("non-group name must yield nil, got %v", got)
	}
	if got := expandGroupMembers("G0010001.JPG", "5", "1", nil); got != nil {
		t.Errorf("inverted range must yield nil, got %v", got)
	}
}
