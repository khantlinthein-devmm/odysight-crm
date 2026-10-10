package cleanerdocs

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/odysight/crm/pkg/sealbox"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	box, err := sealbox.New("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	if err != nil {
		t.Fatal(err)
	}
	return NewStore(t.TempDir(), box, 1)
}

var pngHeader = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

func TestStoreEncryptsAtRest(t *testing.T) {
	s := testStore(t)
	data := append(append([]byte{}, pngHeader...), []byte("PASSPORT-SCAN")...)
	name, err := s.Save(data)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(s.dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("PASSPORT-SCAN")) || bytes.Contains(raw, []byte("PNG")) {
		t.Fatal("file on disk must be encrypted")
	}
	got, err := s.Read(name)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("read back failed: %v", err)
	}
}

func TestStoreRejectsTraversalAndOversize(t *testing.T) {
	s := testStore(t)
	for _, n := range []string{"../x.sealed", "..", "a/b.sealed", "x.png", ""} {
		if _, err := s.Read(n); err == nil {
			t.Errorf("name %q must be rejected", n)
		}
	}
	if _, err := s.Save(make([]byte, 2<<20)); err == nil {
		t.Error("oversized file must be rejected")
	}
}

func TestStoreSweepKeepsReferencedAndFresh(t *testing.T) {
	s := testStore(t)
	keep, _ := s.Save(pngHeader)
	orphan, _ := s.Save(pngHeader)
	fresh, _ := s.Save(pngHeader)
	old := time.Now().Add(-2 * time.Hour)
	_ = os.Chtimes(filepath.Join(s.dir, keep), old, old)
	_ = os.Chtimes(filepath.Join(s.dir, orphan), old, old)
	n, err := s.Sweep(map[string]bool{keep: true}, time.Hour)
	if err != nil || n != 1 {
		t.Fatalf("expected 1 removal, got %d (%v)", n, err)
	}
	for name, want := range map[string]bool{keep: true, orphan: false, fresh: true} {
		_, err := os.Stat(filepath.Join(s.dir, name))
		if (err == nil) != want {
			t.Errorf("%s exists=%v want %v", name, err == nil, want)
		}
	}
}

func TestSniffType(t *testing.T) {
	if ct, _, err := SniffType([]byte("%PDF-1.7\n")); err != nil || ct != "application/pdf" {
		t.Errorf("pdf: %q %v", ct, err)
	}
	if ct, _, err := SniffType(pngHeader); err != nil || ct != "image/png" {
		t.Errorf("png: %q %v", ct, err)
	}
	for _, bad := range [][]byte{[]byte("MZ\x90\x00"), []byte("<html><script>")} {
		if _, _, err := SniffType(bad); err == nil {
			t.Errorf("%q must be rejected", bad)
		}
	}
}

func TestMaskNumber(t *testing.T) {
	cases := map[string]string{"": "", "AB12": "••••", "MA1234567": "MA••••567", "ก1234567": "ก1•••567"}
	for in, want := range cases {
		if got := MaskNumber(in); got != want {
			t.Errorf("MaskNumber(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExpiryStatusAndStages(t *testing.T) {
	now := time.Date(2026, 10, 10, 23, 30, 0, 0, time.Local)
	day := func(d int) *time.Time {
		t := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC).AddDate(0, 0, d)
		return &t
	}
	cases := []struct {
		days   int
		status string
		stage  int16
	}{
		{-1, StatusExpired, 4},
		{0, StatusExpiring, 3},
		{7, StatusExpiring, 3},
		{8, StatusExpiring, 2},
		{30, StatusExpiring, 2},
		{60, StatusExpiring, 1},
		{61, StatusValid, 0},
	}
	for _, c := range cases {
		if got := DaysUntil(*day(c.days), now); got != c.days {
			t.Errorf("DaysUntil(+%d) = %d", c.days, got)
		}
		if got := ExpiryStatus(day(c.days), now); got != c.status {
			t.Errorf("status(+%d) = %s, want %s", c.days, got, c.status)
		}
		if got := reminderStage(c.days); got != c.stage {
			t.Errorf("stage(+%d) = %d, want %d", c.days, got, c.stage)
		}
	}
	if ExpiryStatus(nil, now) != StatusNone {
		t.Error("nil expiry must be none")
	}
}

func TestFieldsParse(t *testing.T) {
	s := func(v string) *string { return &v }
	p, err := FieldsInput{Type: s("passport"), Number: s(" ma123 "), ExpiryDate: s("")}.parse()
	if err != nil || p.typ != TypePassport || p.number != "MA123" || !p.setExpiry || p.expiry != nil {
		t.Fatalf("unexpected parse: %+v %v", p, err)
	}
	if p, err := (FieldsInput{Type: s("resume")}).parse(); err != nil || p.typ != TypeResume {
		t.Errorf("resume must be accepted: %v", err)
	}
	if _, err := (FieldsInput{Type: s("driver")}).parse(); err == nil {
		t.Error("unknown type must be rejected")
	}
	if _, err := (FieldsInput{ExpiryDate: s("10/10/2026")}).parse(); err == nil {
		t.Error("bad date must be rejected")
	}
}

func TestCleanOriginalName(t *testing.T) {
	if got := cleanOriginalName(`C:\scans\"pass"port.pdf`); got != "passport.pdf" {
		t.Errorf("got %q", got)
	}
}
