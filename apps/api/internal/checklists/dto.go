package checklists

import (
	"strings"

	"github.com/odysight/crm/pkg/response"
)

type TemplateItemDTO struct {
	ID        int64  `json:"id"`
	Label     string `json:"label"`
	SortOrder int    `json:"sortOrder"`
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
	IsCompleted    bool    `json:"isCompleted"`
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
	Items             []ChecklistItemDTO `json:"items"`
	CreatedAt         string             `json:"createdAt"`
	UpdatedAt         string             `json:"updatedAt"`
}

func toTemplateDTO(t Template) TemplateDTO {
	items := make([]TemplateItemDTO, 0, len(t.Items))
	for _, it := range t.Items {
		items = append(items, TemplateItemDTO{ID: it.ID, Label: it.Label, SortOrder: it.SortOrder})
	}
	return TemplateDTO{
		ID: t.ID, Name: t.Name, ServiceType: t.ServiceType, IsActive: t.IsActive, Items: items,
		CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toDTO(c Checklist) ChecklistDTO {
	items := make([]ChecklistItemDTO, 0, len(c.Items))
	for _, it := range c.Items {
		var completedAt *string
		if it.CompletedAt != nil {
			v := it.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
			completedAt = &v
		}
		items = append(items, ChecklistItemDTO{
			ID: it.ID, Label: it.Label, IsCompleted: it.IsCompleted,
			CompletedBy: it.CompletedBy, CompletedAt: completedAt, Notes: it.Notes,
			BeforePhotoURL: it.BeforePhotoURL, AfterPhotoURL: it.AfterPhotoURL, SortOrder: it.SortOrder,
		})
	}
	var confirmed *string
	if c.ClientConfirmedAt != nil {
		v := c.ClientConfirmedAt.Format("2006-01-02T15:04:05Z07:00")
		confirmed = &v
	}
	return ChecklistDTO{
		ID: c.ID, BookingID: c.BookingID, TemplateID: c.TemplateID, Status: string(c.Status),
		ClientSignature: c.ClientSignature, ClientConfirmedAt: confirmed, Items: items,
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type CreateTemplateRequest struct {
	Name        string   `json:"name"`
	ServiceType string   `json:"serviceType"`
	Items       []string `json:"items"`
}

func (r *CreateTemplateRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.ServiceType = strings.TrimSpace(r.ServiceType)
	if r.Name == "" {
		return response.NewAPIError(400, "name is required")
	}
	if len(r.Items) == 0 {
		return response.NewAPIError(400, "at least one item is required")
	}
	for i := range r.Items {
		r.Items[i] = strings.TrimSpace(r.Items[i])
		if r.Items[i] == "" {
			return response.NewAPIError(400, "items cannot contain blank labels")
		}
	}
	return nil
}

type CreateChecklistRequest struct {
	BookingID  int64   `json:"bookingId"`
	TemplateID *int64  `json:"templateId"`
	Items      []string `json:"items"`
}

func (r *CreateChecklistRequest) Validate() error {
	if r.BookingID < 1 {
		return response.NewAPIError(400, "bookingId is required")
	}
	if r.TemplateID == nil && len(r.Items) == 0 {
		return response.NewAPIError(400, "templateId or items is required")
	}
	for i := range r.Items {
		r.Items[i] = strings.TrimSpace(r.Items[i])
		if r.Items[i] == "" {
			return response.NewAPIError(400, "items cannot contain blank labels")
		}
	}
	return nil
}

type CompleteItemRequest struct {
	IsCompleted    bool    `json:"isCompleted"`
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
