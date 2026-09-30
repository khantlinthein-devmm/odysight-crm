package checklists

import (
	"context"
	"errors"
	"strings"

	"github.com/odysight/crm/pkg/dberror"
	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListTemplates(ctx context.Context, params pagination.Params) ([]Template, int, error) {
	return s.repo.ListTemplates(ctx, params)
}

func (s *Service) CreateTemplate(ctx context.Context, req CreateTemplateRequest) (Template, error) {
	if err := req.Validate(); err != nil {
		return Template{}, err
	}
	return s.repo.CreateTemplate(ctx, req.Name, req.ServiceType, req.Items)
}

func (s *Service) GetByBooking(ctx context.Context, bookingID int64) (Checklist, error) {
	c, err := s.repo.GetByBooking(ctx, bookingID)
	if err != nil {
		return Checklist{}, mapRepoError(err)
	}
	return c, nil
}

func (s *Service) Create(ctx context.Context, req CreateChecklistRequest) (Checklist, error) {
	if err := req.Validate(); err != nil {
		return Checklist{}, err
	}
	// No template chosen: use the one for the booking's service type.
	if req.TemplateID == nil && len(req.Items) == 0 {
		id, err := s.repo.TemplateForBooking(ctx, req.BookingID)
		if err != nil {
			return Checklist{}, mapRepoError(err)
		}
		req.TemplateID = &id
	}
	c, err := s.repo.Create(ctx, req.BookingID, req.TemplateID, req.Items)
	if err != nil {
		return Checklist{}, mapRepoError(err)
	}
	return c, nil
}

func (s *Service) CompleteItem(ctx context.Context, itemID int64, req CompleteItemRequest) (Checklist, error) {
	var by *string
	if req.CompletedBy != nil {
		v := strings.TrimSpace(*req.CompletedBy)
		by = &v
	}
	var notes *string
	if req.Notes != nil {
		v := strings.TrimSpace(*req.Notes)
		notes = &v
	}
	c, err := s.repo.CompleteItem(ctx, itemID, req.IsCompleted, by, notes, req.BeforePhotoURL, req.AfterPhotoURL)
	if err != nil {
		return Checklist{}, mapRepoError(err)
	}
	return c, nil
}

func (s *Service) Confirm(ctx context.Context, bookingID int64, req ConfirmChecklistRequest) (Checklist, error) {
	if err := req.Validate(); err != nil {
		return Checklist{}, err
	}
	c, err := s.repo.Confirm(ctx, bookingID, req.ClientSignature)
	if err != nil {
		return Checklist{}, mapRepoError(err)
	}
	return c, nil
}

// AttachPhoto records an uploaded photo URL after the file is stored.
func (s *Service) AttachPhoto(ctx context.Context, itemID int64, kind, url string) (Checklist, error) {
	k, err := normalizeKind(kind)
	if err != nil {
		return Checklist{}, err
	}
	c, err := s.repo.SetItemPhoto(ctx, itemID, k, url)
	if err != nil {
		return Checklist{}, mapRepoError(err)
	}
	return c, nil
}

func mapRepoError(err error) error {
	switch {
	case errors.Is(err, ErrTemplateNotFound):
		return response.NewAPIError(422, "that checklist template has no items or does not exist")
	case errors.Is(err, ErrNoTemplate):
		return response.NewAPIError(422, "no checklist template yet: add one under Checklists → Templates")
	case errors.Is(err, ErrAlreadyExists):
		return response.NewAPIError(409, "this booking already has a checklist")
	}
	if errors.Is(err, ErrItemNotFound) {
		return dberror.Map(err, ErrItemNotFound, "checklist item not found")
	}
	return dberror.Map(err, ErrNotFound, "checklist not found")
}
