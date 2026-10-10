package cleanerdocs

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/odysight/crm/pkg/sealbox"
)

// Store keeps scanned documents on local disk, AES-GCM sealed with the stored
// file name as associated data. Files are never served directly: the
// authenticated download endpoint decrypts them in memory.
type Store struct {
	dir      string
	box      *sealbox.Box
	maxBytes int64
}

const fileSuffix = ".sealed"

func NewStore(dir string, box *sealbox.Box, maxMB int) *Store {
	if maxMB <= 0 {
		maxMB = 8
	}
	return &Store{dir: dir, box: box, maxBytes: int64(maxMB) << 20}
}

func (s *Store) MaxBytes() int64 { return s.maxBytes }

// allowedTypes is keyed by the sniffed content type, never the client's
// filename or header.
var allowedTypes = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/webp":      ".webp",
	"application/pdf": ".pdf",
}

// SniffType detects the real content type from the file's leading bytes.
func SniffType(data []byte) (contentType, ext string, err error) {
	ct := http.DetectContentType(data)
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	ext, ok := allowedTypes[ct]
	if !ok {
		return "", "", badRequest("file must be a JPG, PNG, WEBP image or PDF")
	}
	return ct, ext, nil
}

// Save seals data and writes it under a random name, returning that name.
func (s *Store) Save(data []byte) (string, error) {
	if int64(len(data)) > s.maxBytes {
		return "", badRequest(fmt.Sprintf("file exceeds the %d MB limit", s.maxBytes>>20))
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return "", fmt.Errorf("prepare document dir: %w", err)
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("document name: %w", err)
	}
	name := hex.EncodeToString(b) + fileSuffix
	sealed, err := s.box.Seal(data, []byte(name))
	if err != nil {
		return "", err
	}
	dst := filepath.Join(s.dir, name)
	if err := os.WriteFile(dst, sealed, 0o600); err != nil {
		_ = os.Remove(dst)
		return "", fmt.Errorf("store document: %w", err)
	}
	return name, nil
}

func (s *Store) path(name string) (string, error) {
	base := filepath.Base(name)
	if base != name || !strings.HasSuffix(base, fileSuffix) || strings.Contains(base, "\x00") {
		return "", fmt.Errorf("invalid document name")
	}
	return filepath.Join(s.dir, base), nil
}

// Read loads and decrypts a stored document.
func (s *Store) Read(name string) ([]byte, error) {
	p, err := s.path(name)
	if err != nil {
		return nil, err
	}
	sealed, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read document: %w", err)
	}
	return s.box.Open(sealed, []byte(name))
}

// Remove deletes a stored document; a missing file is not an error.
func (s *Store) Remove(name string) error {
	if name == "" {
		return nil
	}
	p, err := s.path(name)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Sweep removes sealed files no longer referenced by any row (e.g. after a
// cleaner was deleted and its rows cascaded). Files younger than grace are
// kept so an in-flight upload is never raced.
func (s *Store) Sweep(referenced map[string]bool, grace time.Duration) (int, error) {
	entries, err := os.ReadDir(s.dir)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	cutoff := time.Now().Add(-grace)
	removed := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, fileSuffix) || referenced[name] {
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(s.dir, name)); err == nil {
			removed++
		}
	}
	return removed, nil
}
