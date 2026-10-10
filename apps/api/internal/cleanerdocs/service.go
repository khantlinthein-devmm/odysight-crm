package cleanerdocs

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/odysight/crm/pkg/dberror"
	"github.com/odysight/crm/pkg/response"
	"github.com/odysight/crm/pkg/sealbox"
)

// Service is nil-safe on its crypto: with no DOCUMENTS_ENC_KEY configured the
// feature answers 503 instead of ever storing documents in the clear.
type Service struct {
	repo  *Repository
	box   *sealbox.Box
	store *Store
	now   func() time.Time
}

func NewService(repo *Repository, box *sealbox.Box, store *Store) *Service {
	return &Service{repo: repo, box: box, store: store, now: time.Now}
}

func (s *Service) Enabled() bool { return s.box != nil && s.store != nil }

var errDisabled = response.NewAPIError(503, "document storage is not configured (set DOCUMENTS_ENC_KEY)")

func mapRepoError(err error) error {
	if err == ErrCleanerNotFound {
		return response.NewAPIError(404, "cleaner not found")
	}
	return dberror.Map(err, ErrNotFound, "document not found")
}

// numberAAD binds a sealed document number to its cleaner so ciphertexts
// cannot be moved between cleaners.
func numberAAD(cleanerID int64) []byte {
	return []byte(fmt.Sprintf("cleaner_documents.number:%d", cleanerID))
}

func (s *Service) sealNumber(cleanerID int64, number string) ([]byte, error) {
	if number == "" {
		return nil, nil
	}
	return s.box.Seal([]byte(number), numberAAD(cleanerID))
}

func (s *Service) openNumber(d Document) string {
	if len(d.NumberEnc) == 0 {
		return ""
	}
	plain, err := s.box.Open(d.NumberEnc, numberAAD(d.CleanerID))
	if err != nil {
		slog.Warn("cleaner document number could not be decrypted", "document", d.ID)
		return ""
	}
	return string(plain)
}

func (s *Service) requireCleaner(ctx context.Context, cleanerID int64) error {
	ok, err := s.repo.CleanerExists(ctx, cleanerID)
	if err != nil {
		return err
	}
	if !ok {
		return mapRepoError(ErrCleanerNotFound)
	}
	return nil
}

// List returns a cleaner's documents with masked numbers.
func (s *Service) List(ctx context.Context, cleanerID int64) ([]DocumentDTO, error) {
	if !s.Enabled() {
		return nil, errDisabled
	}
	if err := s.requireCleaner(ctx, cleanerID); err != nil {
		return nil, err
	}
	docs, err := s.repo.ListByCleaner(ctx, cleanerID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	out := make([]DocumentDTO, 0, len(docs))
	for _, d := range docs {
		out = append(out, toDTO(d, s.openNumber(d), true, now))
	}
	return out, nil
}

// Statuses returns type + expiry only; it needs no key and is safe for
// roles without document access.
func (s *Service) Statuses(ctx context.Context, cleanerID int64) ([]StatusDTO, error) {
	if err := s.requireCleaner(ctx, cleanerID); err != nil {
		return nil, err
	}
	docs, err := s.repo.ListByCleaner(ctx, cleanerID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	out := make([]StatusDTO, 0, len(docs))
	for _, d := range docs {
		out = append(out, toStatusDTO(d, now))
	}
	return out, nil
}

// Reveal returns one document with its full number and audits the access.
func (s *Service) Reveal(ctx context.Context, id, userID int64) (DocumentDTO, error) {
	if !s.Enabled() {
		return DocumentDTO{}, errDisabled
	}
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return DocumentDTO{}, mapRepoError(err)
	}
	s.audit(ctx, userID, "VIEW", id)
	return toDTO(d, s.openNumber(d), false, s.now()), nil
}

// Upload is the decoded multipart file of a create request.
type Upload struct {
	Data         []byte
	OriginalName string
}

func cleanOriginalName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '"' {
			return -1
		}
		return r
	}, name)
	if utf8.RuneCountInString(name) > 120 {
		name = string([]rune(name)[:120])
	}
	if name == "." || name == "/" {
		return ""
	}
	return name
}

func (s *Service) Create(ctx context.Context, cleanerID, userID int64, in FieldsInput, file *Upload) (DocumentDTO, error) {
	if !s.Enabled() {
		return DocumentDTO{}, errDisabled
	}
	p, err := in.parse()
	if err != nil {
		return DocumentDTO{}, err
	}
	if !p.setType {
		return DocumentDTO{}, badRequest("type is required")
	}
	if p.issue != nil && p.expiry != nil && p.expiry.Before(*p.issue) {
		return DocumentDTO{}, badRequest("expiryDate must be after issueDate")
	}
	if err := s.requireCleaner(ctx, cleanerID); err != nil {
		return DocumentDTO{}, err
	}

	d := Document{
		CleanerID:  cleanerID,
		Type:       p.typ,
		IssueDate:  p.issue,
		ExpiryDate: p.expiry,
		Notes:      p.notes,
		UploadedBy: &userID,
	}
	if d.NumberEnc, err = s.sealNumber(cleanerID, p.number); err != nil {
		return DocumentDTO{}, err
	}
	if file != nil && len(file.Data) > 0 {
		ct, _, err := SniffType(file.Data)
		if err != nil {
			return DocumentDTO{}, err
		}
		name, err := s.store.Save(file.Data)
		if err != nil {
			return DocumentDTO{}, err
		}
		d.FileName, d.ContentType, d.SizeBytes = name, ct, int64(len(file.Data))
		d.OriginalName = cleanOriginalName(file.OriginalName)
	}
	if p.number == "" && d.FileName == "" && d.ExpiryDate == nil {
		return DocumentDTO{}, badRequest("provide a document number, expiry date or file")
	}

	created, err := s.repo.Create(ctx, d)
	if err != nil {
		_ = s.store.Remove(d.FileName)
		return DocumentDTO{}, mapRepoError(err)
	}
	return toDTO(created, p.number, true, s.now()), nil
}

func (s *Service) Update(ctx context.Context, id int64, in FieldsInput) (DocumentDTO, error) {
	if !s.Enabled() {
		return DocumentDTO{}, errDisabled
	}
	p, err := in.parse()
	if err != nil {
		return DocumentDTO{}, err
	}
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return DocumentDTO{}, mapRepoError(err)
	}
	number := s.openNumber(d)
	if p.setType {
		d.Type = p.typ
	}
	if p.setNumber {
		number = p.number
		if d.NumberEnc, err = s.sealNumber(d.CleanerID, number); err != nil {
			return DocumentDTO{}, err
		}
	}
	if p.setIssue {
		d.IssueDate = p.issue
	}
	if p.setExpiry {
		if !sameDate(d.ExpiryDate, p.expiry) {
			// A renewed document starts the reminder ladder again.
			d.ReminderStage = 0
		}
		d.ExpiryDate = p.expiry
	}
	if p.setNotes {
		d.Notes = p.notes
	}
	if d.IssueDate != nil && d.ExpiryDate != nil && d.ExpiryDate.Before(*d.IssueDate) {
		return DocumentDTO{}, badRequest("expiryDate must be after issueDate")
	}
	updated, err := s.repo.Update(ctx, d)
	if err != nil {
		return DocumentDTO{}, mapRepoError(err)
	}
	return toDTO(updated, number, true, s.now()), nil
}

func sameDate(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Format(dateLayout) == b.Format(dateLayout)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if !s.Enabled() {
		return errDisabled
	}
	fileName, err := s.repo.Delete(ctx, id)
	if err != nil {
		return mapRepoError(err)
	}
	if err := s.store.Remove(fileName); err != nil {
		// The orphan sweep in the reminder runner retries this later.
		slog.Warn("cleaner document file removal failed", "document", id, "error", err)
	}
	return nil
}

// File decrypts a document's scan for download and audits the access.
func (s *Service) File(ctx context.Context, id, userID int64) (Document, []byte, error) {
	if !s.Enabled() {
		return Document{}, nil, errDisabled
	}
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return Document{}, nil, mapRepoError(err)
	}
	if d.FileName == "" {
		return Document{}, nil, response.NewAPIError(404, "document has no file")
	}
	data, err := s.store.Read(d.FileName)
	if err != nil {
		return Document{}, nil, fmt.Errorf("open document %d: %w", id, err)
	}
	s.audit(ctx, userID, "DOWNLOAD", id)
	return d, data, nil
}

func (s *Service) Expiring(ctx context.Context, withinDays int) ([]ExpiringDTO, error) {
	if withinDays < 0 || withinDays > 365 {
		withinDays = ExpiringWindowDays
	}
	now := s.now()
	items, err := s.repo.Expiring(ctx, today(now).AddDate(0, 0, withinDays))
	if err != nil {
		return nil, err
	}
	out := make([]ExpiringDTO, 0, len(items))
	for _, e := range items {
		out = append(out, toExpiringDTO(e, now))
	}
	return out, nil
}

func (s *Service) audit(ctx context.Context, userID int64, action string, id int64) {
	resource := fmt.Sprintf("/api/v1/cleaner-documents/%d", id)
	if err := s.repo.LogAccess(ctx, userID, action, resource, id); err != nil {
		slog.Warn("cleaner document audit write failed", "document", id, "error", err)
	}
}
