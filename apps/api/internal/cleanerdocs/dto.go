package cleanerdocs

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/odysight/crm/pkg/response"
)

const dateLayout = "2006-01-02"

// DocumentDTO is returned to holders of cleaner_documents.read. Number is
// masked in list responses and only revealed by the single-document GET,
// which is written to the audit log.
type DocumentDTO struct {
	ID           int64     `json:"id"`
	CleanerID    int64     `json:"cleanerId"`
	Type         DocType   `json:"type"`
	Number       string    `json:"number"`
	NumberMasked bool      `json:"numberMasked"`
	IssueDate    *string   `json:"issueDate"`
	ExpiryDate   *string   `json:"expiryDate"`
	ExpiryStatus string    `json:"expiryStatus"`
	DaysToExpiry *int      `json:"daysToExpiry"`
	Notes        string    `json:"notes"`
	HasFile      bool      `json:"hasFile"`
	OriginalName string    `json:"originalName"`
	ContentType  string    `json:"contentType"`
	SizeBytes    int64     `json:"sizeBytes"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// StatusDTO is the non-sensitive view (type + expiry only) for staff who can
// see cleaners but not their documents, e.g. dispatch.
type StatusDTO struct {
	ID           int64   `json:"id"`
	Type         DocType `json:"type"`
	ExpiryDate   *string `json:"expiryDate"`
	ExpiryStatus string  `json:"expiryStatus"`
	DaysToExpiry *int    `json:"daysToExpiry"`
}

type ExpiringDTO struct {
	DocumentID   int64   `json:"documentId"`
	CleanerID    int64   `json:"cleanerId"`
	CleanerName  string  `json:"cleanerName"`
	Type         DocType `json:"type"`
	ExpiryDate   string  `json:"expiryDate"`
	ExpiryStatus string  `json:"expiryStatus"`
	DaysToExpiry int     `json:"daysToExpiry"`
}

func formatDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(dateLayout)
	return &s
}

func daysPtr(t *time.Time, now time.Time) *int {
	if t == nil {
		return nil
	}
	d := DaysUntil(*t, now)
	return &d
}

func toDTO(d Document, number string, masked bool, now time.Time) DocumentDTO {
	if masked {
		number = MaskNumber(number)
	}
	return DocumentDTO{
		ID:           d.ID,
		CleanerID:    d.CleanerID,
		Type:         d.Type,
		Number:       number,
		NumberMasked: masked,
		IssueDate:    formatDate(d.IssueDate),
		ExpiryDate:   formatDate(d.ExpiryDate),
		ExpiryStatus: ExpiryStatus(d.ExpiryDate, now),
		DaysToExpiry: daysPtr(d.ExpiryDate, now),
		Notes:        d.Notes,
		HasFile:      d.FileName != "",
		OriginalName: d.OriginalName,
		ContentType:  d.ContentType,
		SizeBytes:    d.SizeBytes,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

func toStatusDTO(d Document, now time.Time) StatusDTO {
	return StatusDTO{
		ID:           d.ID,
		Type:         d.Type,
		ExpiryDate:   formatDate(d.ExpiryDate),
		ExpiryStatus: ExpiryStatus(d.ExpiryDate, now),
		DaysToExpiry: daysPtr(d.ExpiryDate, now),
	}
}

func toExpiringDTO(e ExpiringItem, now time.Time) ExpiringDTO {
	exp := e.ExpiryDate
	return ExpiringDTO{
		DocumentID:   e.DocumentID,
		CleanerID:    e.CleanerID,
		CleanerName:  e.CleanerName,
		Type:         e.Type,
		ExpiryDate:   exp.Format(dateLayout),
		ExpiryStatus: ExpiryStatus(&exp, now),
		DaysToExpiry: DaysUntil(exp, now),
	}
}

// MaskNumber keeps the first two and last three characters, e.g.
// "MA1234567" → "MA••••567". Short values are fully masked.
func MaskNumber(s string) string {
	r := []rune(s)
	n := len(r)
	if n == 0 {
		return ""
	}
	if n <= 6 {
		return strings.Repeat("•", n)
	}
	return string(r[:2]) + strings.Repeat("•", n-5) + string(r[n-3:])
}

// Fields shared by the multipart create form and the JSON update body.
// Dates use YYYY-MM-DD; an empty string clears the value on update.
type FieldsInput struct {
	Type       *string `json:"type"`
	Number     *string `json:"number"`
	IssueDate  *string `json:"issueDate"`
	ExpiryDate *string `json:"expiryDate"`
	Notes      *string `json:"notes"`
}

// parsed is the validated form of FieldsInput. set* flags mark which fields
// the caller supplied.
type parsed struct {
	typ       DocType
	setType   bool
	number    string
	setNumber bool
	issue     *time.Time
	setIssue  bool
	expiry    *time.Time
	setExpiry bool
	notes     string
	setNotes  bool
}

func badRequest(msg string) error { return response.NewAPIError(400, msg) }

func parseDate(field, v string) (*time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(dateLayout, v)
	if err != nil {
		return nil, badRequest(field + " must be YYYY-MM-DD")
	}
	return &t, nil
}

func (in FieldsInput) parse() (parsed, error) {
	var p parsed
	if in.Type != nil {
		p.typ, p.setType = DocType(strings.TrimSpace(*in.Type)), true
		if !p.typ.Valid() {
			return p, badRequest("type must be one of passport, visa, work_permit, pink_card, id_card, resume, other")
		}
	}
	if in.Number != nil {
		p.number, p.setNumber = strings.ToUpper(strings.TrimSpace(*in.Number)), true
		if utf8.RuneCountInString(p.number) > 64 {
			return p, badRequest("number must be at most 64 characters")
		}
	}
	var err error
	if in.IssueDate != nil {
		if p.issue, err = parseDate("issueDate", *in.IssueDate); err != nil {
			return p, err
		}
		p.setIssue = true
	}
	if in.ExpiryDate != nil {
		if p.expiry, err = parseDate("expiryDate", *in.ExpiryDate); err != nil {
			return p, err
		}
		p.setExpiry = true
	}
	if in.Notes != nil {
		p.notes, p.setNotes = strings.TrimSpace(*in.Notes), true
		if utf8.RuneCountInString(p.notes) > 1000 {
			return p, badRequest("notes must be at most 1000 characters")
		}
	}
	return p, nil
}
