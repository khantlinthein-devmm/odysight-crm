package checklists

import (
	"mime/multipart"
	"testing"
)

func TestNormalizeKind(t *testing.T) {
	for _, k := range []string{"before", "after", " Before ", "AFTER"} {
		if _, err := normalizeKind(k); err != nil {
			t.Errorf("kind %q rejected: %v", k, err)
		}
	}
	if _, err := normalizeKind("during"); err == nil {
		t.Error("unknown kind must be rejected")
	}
}

func TestPhotoStoreRejectsTraversal(t *testing.T) {
	s := NewPhotoStore(t.TempDir(), 8)
	for _, name := range []string{"../secret", "..", "x.exe", "photo.gif", ""} {
		if _, _, err := s.Open(name); err == nil {
			t.Errorf("photo name %q must be rejected", name)
		}
	}
}

func TestPhotoStoreRejectsBadExtension(t *testing.T) {
	s := NewPhotoStore(t.TempDir(), 8)
	header := &multipart.FileHeader{Filename: "evil.exe", Size: 10}
	if _, err := s.Save(1, "before", nil, header); err == nil {
		t.Error("exe upload must be rejected")
	}
	header = &multipart.FileHeader{Filename: "ok.png", Size: 10}
	if _, err := s.Save(1, "during", nil, header); err == nil {
		t.Error("bad kind must be rejected before touching storage")
	}
}
