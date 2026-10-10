package chat

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// FileStore keeps chat voice messages on local disk (and serves photos sent
// before photos were switched off, until the cleanup removes them). Files are
// never served publicly: downloads go through an endpoint that checks the
// caller is in the conversation.
type FileStore struct {
	dir string
}

func NewFileStore(uploadDir string) *FileStore {
	return &FileStore{dir: filepath.Join(uploadDir, "chat")}
}

// A 1-minute note at 24 kbps is ~180 KB; the cap leaves room for browsers
// that ignore the requested bitrate.
const maxVoiceBytes = 3 << 20

// Accepted types, keyed by what the content sniffs as. Voice notes come from
// the browser's MediaRecorder: webm/ogg (Chrome, Firefox, Android) or mp4/aac
// (iPhone Safari).

var voiceTypes = map[string]string{
	"video/webm": ".webm", "audio/webm": ".webm",
	"application/ogg": ".ogg", "audio/ogg": ".ogg",
	"video/mp4": ".m4a", "audio/mp4": ".m4a", "audio/aac": ".aac", "audio/mpeg": ".mp3",
	"audio/wave": ".wav",
}

// servedType maps our stored extension to the Content-Type we serve.
var servedType = map[string]string{
	".jpg": "image/jpeg", ".png": "image/png", ".webp": "image/webp",
	".webm": "audio/webm", ".ogg": "audio/ogg", ".m4a": "audio/mp4", ".aac": "audio/aac",
	".mp3": "audio/mpeg", ".wav": "audio/wav",
}

// Save validates and stores an upload, returning the stored file name and
// the Content-Type it will be served with. The type is decided by sniffing
// the bytes, never by the client's filename or header.
func (s *FileStore) Save(kind string, r io.Reader) (string, string, error) {
	if kind != KindVoice {
		return "", "", fmt.Errorf("only voice messages can be attached")
	}
	limit := int64(maxVoiceBytes)
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return "", "", fmt.Errorf("read upload: %w", err)
	}
	if int64(len(data)) > limit {
		return "", "", fmt.Errorf("file is larger than %d MB", limit>>20)
	}
	if len(data) == 0 {
		return "", "", fmt.Errorf("file is empty")
	}
	sniffed := http.DetectContentType(data)
	if i := strings.IndexByte(sniffed, ';'); i >= 0 {
		sniffed = sniffed[:i]
	}
	ext, ok := voiceTypes[sniffed]
	if !ok && isMP4Audio(data) {
		ext, ok = ".m4a", true
	}
	if !ok {
		return "", "", fmt.Errorf("unsupported voice format")
	}
	if err := os.MkdirAll(s.dir, 0o750); err != nil {
		return "", "", fmt.Errorf("prepare chat upload dir: %w", err)
	}
	var rnd [16]byte
	if _, err := rand.Read(rnd[:]); err != nil {
		return "", "", err
	}
	name := hex.EncodeToString(rnd[:]) + ext
	if err := os.WriteFile(filepath.Join(s.dir, name), data, 0o640); err != nil {
		return "", "", fmt.Errorf("store chat file: %w", err)
	}
	return name, servedType[ext], nil
}

// isMP4Audio recognises an ISO-BMFF file ("ftyp" box) that the sniffer does
// not label, as Safari's M4A voice notes sometimes are.
func isMP4Audio(data []byte) bool {
	return len(data) > 12 && bytes.Equal(data[4:8], []byte("ftyp"))
}

// Path resolves a stored file name, refusing anything that is not a bare
// name we generated.
func (s *FileStore) Path(name string) (string, string, bool) {
	if name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
		return "", "", false
	}
	ct, ok := servedType[filepath.Ext(name)]
	if !ok {
		return "", "", false
	}
	return filepath.Join(s.dir, name), ct, true
}

// Remove deletes a stored file; a file that is already gone is not an error.
func (s *FileStore) Remove(name string) error {
	path, _, ok := s.Path(name)
	if !ok {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
