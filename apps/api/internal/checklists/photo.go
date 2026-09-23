package checklists

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PhotoStore persists proof-of-work photos on local disk. Files are never
// served publicly: downloads go through the authenticated GET photo endpoint.
type PhotoStore struct {
	dir      string
	maxBytes int64
}

func NewPhotoStore(dir string, maxMB int) *PhotoStore {
	if maxMB <= 0 {
		maxMB = 8
	}
	return &PhotoStore{dir: dir, maxBytes: int64(maxMB) << 20}
}

var allowedPhotoExt = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
}

func normalizeKind(kind string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "before":
		return "before", nil
	case "after":
		return "after", nil
	}
	return "", fmt.Errorf("kind must be before or after")
}

// Save stores an uploaded photo for a checklist item and returns the
// API-relative URL recorded on the item.
func (s *PhotoStore) Save(itemID int64, kind string, file multipart.File, header *multipart.FileHeader) (string, error) {
	kind, err := normalizeKind(kind)
	if err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	contentType, ok := allowedPhotoExt[ext]
	if !ok {
		return "", fmt.Errorf("photo must be jpg, png or webp")
	}
	if header.Size > s.maxBytes {
		return "", fmt.Errorf("photo exceeds the %d MB limit", s.maxBytes>>20)
	}
	if err := os.MkdirAll(s.dir, 0o750); err != nil {
		return "", fmt.Errorf("prepare upload dir: %w", err)
	}
	name := fmt.Sprintf("item%d_%s_%d%s", itemID, kind, time.Now().UnixNano(), ext)
	dst := filepath.Join(s.dir, name)
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return "", fmt.Errorf("store photo: %w", err)
	}
	defer func() { _ = out.Close() }()
	src := io.LimitReader(file, s.maxBytes+1)
	n, err := io.Copy(out, src)
	if err != nil {
		_ = os.Remove(dst)
		return "", fmt.Errorf("store photo: %w", err)
	}
	if n > s.maxBytes {
		_ = os.Remove(dst)
		return "", fmt.Errorf("photo exceeds the %d MB limit", s.maxBytes>>20)
	}
	_ = contentType
	return "/api/v1/checklists/photos/" + name, nil
}

// Open returns a stored photo for the authenticated download endpoint. The
// name is reduced to its base element so `../` traversal can never escape
// the upload directory.
func (s *PhotoStore) Open(name string) (*os.File, string, error) {
	base := filepath.Base(name)
	if base == "." || base == "/" || strings.Contains(base, "\x00") {
		return nil, "", fmt.Errorf("invalid photo name")
	}
	ext := strings.ToLower(filepath.Ext(base))
	contentType, ok := allowedPhotoExt[ext]
	if !ok {
		return nil, "", fmt.Errorf("photo not found")
	}
	dst := filepath.Join(s.dir, base)
	rel, err := filepath.Rel(s.dir, dst)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil, "", fmt.Errorf("invalid photo name")
	}
	f, err := os.Open(dst)
	if err != nil {
		return nil, "", fmt.Errorf("photo not found")
	}
	return f, contentType, nil
}
