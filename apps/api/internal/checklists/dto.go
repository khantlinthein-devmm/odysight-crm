package checklists

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
)

const timeLayout = "2006-01-02T15:04:05Z07:00"

func fmtTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	v := t.Format(timeLayout)
	return &v
}

type TemplateItemDTO struct {
	ID           int64  `json:"id"`
	Label        string `json:"label"`
	ItemType     string `json:"itemType"`
	RequirePhoto bool   `json:"requirePhoto"`
	SortOrder    int    `json:"sortOrder"`
}

type TemplateDTO struct {
	ID          int64             `json:"id"`
	Name        string            `json:"name"`
	ServiceType string            `json:"serviceType"`
	IsActive    bool              `json:"isActive"`
	Items       []TemplateItemDTO `json:"items"`
	CreatedAt   string            `json:"createdAt"`
	UpdatedAt   string            `json:"updatedAt"`
}

type ChecklistItemDTO struct {
	ID             int64   `json:"id"`
	Label          string  `json:"label"`
	ItemType       string  `json:"itemType"`
	RequirePhoto   bool    `json:"requirePhoto"`
	IsCompleted    bool    `json:"isCompleted"`
	Score          *int    `json:"score"`
	Result         *string `json:"result"`
	CompletedBy    *string `json:"completedBy"`
	CompletedAt    *string `json:"completedAt"`
	Notes          string  `json:"notes"`
	BeforePhotoURL *string `json:"beforePhotoUrl"`
	AfterPhotoURL  *string `json:"afterPhotoUrl"`
	SortOrder      int     `json:"sortOrder"`
}

type ChecklistDTO struct {
	ID                int64              `json:"id"`
	BookingID         int64              `json:"bookingId"`
	TemplateID        *int64             `json:"templateId"`
	Status            string             `json:"status"`
	ClientSignature   *string            `json:"clientSignature"`
	ClientConfirmedAt *string            `json:"clientConfirmedAt"`
	SubmittedAt       *string            `json:"submittedAt"`
	SubmittedBy       *string            `json:"submittedBy"`
	ReviewStatus      string             `json:"reviewStatus"`
	ReviewNote        string             `json:"reviewNote"`
	ReviewedBy        *string            `json:"reviewedBy"`
	ReviewedAt        *string            `json:"reviewedAt"`
	Locked            bool               `json:"locked"`
	AvgScore          *float64           `json:"avgScore"`
	Items             []ChecklistItemDTO `json:"items"`
	CreatedAt         string             `json:"createdAt"`
	UpdatedAt         string             `json:"updatedAt"`
}

type SummaryDTO struct {
	ID            int64    `json:"id"`
	BookingID     int64    `json:"bookingId"`
	BookingNumber string   `json:"bookingNumber"`
	CustomerName  string   `json:"customerName"`
	ServiceType   string   `json:"serviceType"`
	ScheduledFor  string   `json:"scheduledFor"`
	Cleaners      string   `json:"cleaners"`
	Status        string   `json:"status"`
	SubmittedAt   *string  `json:"submittedAt"`
	ReviewStatus  string   `json:"reviewStatus"`
	ItemsTotal    int      `json:"itemsTotal"`
	ItemsDone     int      `json:"itemsDone"`
	AvgScore      *float64 `json:"avgScore"`
	FailCount     int      `json:"failCount"`
	PhotoCount    int      `json:"photoCount"`
}

type StatusDTO struct {
	BookingID    int64  `json:"bookingId"`
	Status       string `json:"status"`
	Submitted    bool   `json:"submitted"`
	ReviewStatus string `json:"reviewStatus"`
	ReviewNote   string `json:"reviewNote"`
	ItemsTotal   int    `json:"itemsTotal"`
	ItemsDone    int    `json:"itemsDone"`
}

func toTemplateDTO(t Template) TemplateDTO {
	items := make([]TemplateItemDTO, 0, len(t.Items))
	for _, it := range t.Items {
		items = append(items, TemplateItemDTO{ID: it.ID, Label: it.Label, ItemType: string(it.ItemType), RequirePhoto: it.RequirePhoto, SortOrder: it.SortOrder})
	}
	return TemplateDTO{
		ID: t.ID, Name: t.Name, ServiceType: t.ServiceType, IsActive: t.IsActive, Items: items,
		CreatedAt: t.CreatedAt.Format(timeLayout),
		UpdatedAt: t.UpdatedAt.Format(timeLayout),
	}
}

// averageScore is the mean of the answered 1-5 score items, nil when none.
func averageScore(items []ChecklistItem) *float64 {
	sum, n := 0, 0
	for _, it := range items {
		if it.Score != nil {
			sum += *it.Score
			n++
		}
	}
	if n == 0 {
		return nil
	}
	v := float64(sum) / float64(n)
	return &v
}

func toDTO(c Checklist) ChecklistDTO {
	items := make([]ChecklistItemDTO, 0, len(c.Items))
	for _, it := range c.Items {
		items = append(items, ChecklistItemDTO{
			ID: it.ID, Label: it.Label, ItemType: string(it.ItemType), RequirePhoto: it.RequirePhoto,
			IsCompleted: it.IsCompleted, Score: it.Score, Result: it.Result,
			CompletedBy: it.CompletedBy, CompletedAt: fmtTime(it.CompletedAt), Notes: it.Notes,
			BeforePhotoURL: it.BeforePhotoURL, AfterPhotoURL: it.AfterPhotoURL, SortOrder: it.SortOrder,
		})
	}
	return ChecklistDTO{
		ID: c.ID, BookingID: c.BookingID, TemplateID: c.TemplateID, Status: string(c.Status),
		ClientSignature: c.ClientSignature, ClientConfirmedAt: fmtTime(c.ClientConfirmedAt),
		SubmittedAt: fmtTime(c.SubmittedAt), SubmittedBy: c.SubmittedBy,
		ReviewStatus: c.ReviewStatus, ReviewNote: c.ReviewNote, ReviewedBy: c.ReviewedBy, ReviewedAt: fmtTime(c.ReviewedAt),
		Locked: c.Locked(), AvgScore: averageScore(c.Items), Items: items,
		CreatedAt: c.CreatedAt.Format(timeLayout),
		UpdatedAt: c.UpdatedAt.Format(timeLayout),
	}
}

func toSummaryDTO(s Summary) SummaryDTO {
	return SummaryDTO{
		ID: s.ID, BookingID: s.BookingID, BookingNumber: s.BookingNumber, CustomerName: s.CustomerName,
		ServiceType: s.ServiceType, ScheduledFor: s.ScheduledFor.Format(timeLayout), Cleaners: s.Cleaners,
		Status: string(s.Status), SubmittedAt: fmtTime(s.SubmittedAt), ReviewStatus: s.ReviewStatus,
		ItemsTotal: s.ItemsTotal, ItemsDone: s.ItemsDone, AvgScore: s.AvgScore, FailCount: s.FailCount, PhotoCount: s.PhotoCount,
	}
}

// TemplateItemInput accepts either a plain label string (the original API)
// or {"label", "itemType", "requirePhoto"}.
type TemplateItemInput struct {
	Label        string `json:"label"`
	ItemType     string `json:"itemType"`
	RequirePhoto bool   `json:"requirePhoto"`
}

func (in *TemplateItemInput) UnmarshalJSON(b []byte) error {
	if s := bytes.TrimSpace(b); len(s) > 0 && s[0] == '"' {
		return json.Unmarshal(s, &in.Label)
	}
	type plain TemplateItemInput
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*in = TemplateItemInput(p)
	return nil
}

func validateItems(in []TemplateItemInput) ([]TemplateItem, error) {
	if len(in) == 0 {
		return nil, response.NewAPIError(400, "at least one item is required")
	}
	out := make([]TemplateItem, 0, len(in))
	for _, it := range in {
		label := strings.TrimSpace(it.Label)
		if label == "" {
			return nil, response.NewAPIError(400, "items cannot contain blank labels")
		}
		typ := ItemType(strings.TrimSpace(it.ItemType))
		if typ == "" {
			typ = ItemCheck
		}
		if !typ.Valid() {
			return nil, response.NewAPIError(400, "itemType must be check, score or pass_fail")
		}
		out = append(out, TemplateItem{Label: label, ItemType: typ, RequirePhoto: it.RequirePhoto})
	}
	return out, nil
}

type CreateTemplateRequest struct {
	Name        string              `json:"name"`
	ServiceType string              `json:"serviceType"`
	Items       []TemplateItemInput `json:"items"`

	items []TemplateItem
}

func (r *CreateTemplateRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.ServiceType = strings.TrimSpace(r.ServiceType)
	if r.Name == "" {
		return response.NewAPIError(400, "name is required")
	}
	items, err := validateItems(r.Items)
	if err != nil {
		return err
	}
	r.items = items
	return nil
}

type UpdateTemplateRequest struct {
	Name        string              `json:"name"`
	ServiceType string              `json:"serviceType"`
	IsActive    *bool               `json:"isActive"`
	Items       []TemplateItemInput `json:"items"`

	items []TemplateItem
}

func (r *UpdateTemplateRequest) Validate() error {
	c := CreateTemplateRequest{Name: r.Name, ServiceType: r.ServiceType, Items: r.Items}
	if err := c.Validate(); err != nil {
		return err
	}
	r.Name, r.ServiceType, r.items = c.Name, c.ServiceType, c.items
	return nil
}

type CreateChecklistRequest struct {
	BookingID  int64               `json:"bookingId"`
	TemplateID *int64              `json:"templateId"`
	Items      []TemplateItemInput `json:"items"`

	items []TemplateItem
}

func (r *CreateChecklistRequest) Validate() error {
	if r.BookingID < 1 {
		return response.NewAPIError(400, "bookingId is required")
	}
	if len(r.Items) > 0 {
		items, err := validateItems(r.Items)
		if err != nil {
			return err
		}
		r.items = items
	}
	return nil
}

// CompleteItemRequest updates one item. isCompleted drives tick items; score
// (1-5, 0 clears) and result (pass|fail|na, "" clears) drive the other types.
type CompleteItemRequest struct {
	IsCompleted    bool    `json:"isCompleted"`
	Score          *int    `json:"score"`
	Result         *string `json:"result"`
	CompletedBy    *string `json:"completedBy"`
	Notes          *string `json:"notes"`
	BeforePhotoURL *string `json:"beforePhotoUrl"`
	AfterPhotoURL  *string `json:"afterPhotoUrl"`
}

type ConfirmChecklistRequest struct {
	ClientSignature string `json:"clientSignature"`
}

func (r *ConfirmChecklistRequest) Validate() error {
	r.ClientSignature = strings.TrimSpace(r.ClientSignature)
	if r.ClientSignature == "" {
		return response.NewAPIError(400, "clientSignature is required")
	}
	return nil
}

type ReviewRequest struct {
	Decision string `json:"decision"` // approved | rework
	Note     string `json:"note"`
}

func (r *ReviewRequest) Validate() error {
	r.Decision = strings.TrimSpace(r.Decision)
	r.Note = strings.TrimSpace(r.Note)
	switch r.Decision {
	case ReviewApproved:
	case ReviewRework:
		if r.Note == "" {
			return response.NewAPIError(400, "say what needs redoing in the note")
		}
	default:
		return response.NewAPIError(400, "decision must be approved or rework")
	}
	return nil
}
