package wifi

import (
	"reflect"
	"testing"
)

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
