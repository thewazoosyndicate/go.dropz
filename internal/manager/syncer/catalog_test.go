package syncer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dropz/dropz/internal/wifi"
)

// Duplicated camera names resolve local state under their prefixed local
// name; a bare-name lookup would mark both copies downloaded at once.
func TestReadCatalogDuplicateNames(t *testing.T) {
	folder := t.TempDir()
	now := time.Now()
	files := []wifi.MediaFile{
		{Name: "GX010002.MP4", CameraPath: "101GOPRO/GX010002.MP4", Size: 1, CreatedAt: now},
		{Name: "GX010002.MP4", CameraPath: "102GOPRO/GX010002.MP4", Size: 2, CreatedAt: now},
		{Name: "GX010003.MP4", CameraPath: "101GOPRO/GX010003.MP4", Size: 3, CreatedAt: now},
	}
	if err := WriteCatalog(folder, files); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "102GOPRO_GX010002.MP4"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "GX010003.MP4"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	items, _, err := ReadCatalog(folder)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("items = %d, want 3", len(items))
	}
	downloaded := map[string]bool{}
	for _, item := range items {
		downloaded[item.CameraPath] = item.Downloaded
	}
	if downloaded["101GOPRO/GX010002.MP4"] {
		t.Error("101GOPRO copy must not count as downloaded")
	}
	if !downloaded["102GOPRO/GX010002.MP4"] {
		t.Error("102GOPRO copy is on disk under its local name")
	}
	if !downloaded["101GOPRO/GX010003.MP4"] {
		t.Error("unique name keeps its bare local name")
	}
}
