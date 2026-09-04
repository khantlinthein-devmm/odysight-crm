package visacases

import (
	"context"
	"errors"
	"strings"

	"github.com/odysight/crm/pkg/response"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]VisaCase, error) {
	return s.repo.List(ctx)
}

func (s *Service) Get(ctx context.Context, id int64) (VisaCase, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return VisaCase{}, mapRepoError(err)
	}
	return c, nil
}

func (s *Service) Create(ctx context.Context, req CreateVisaCaseRequest) (VisaCase, error) {
	if err := req.Validate(); err != nil {
		return VisaCase{}, err
	}

	c := VisaCase{
		ApplicantName: req.ApplicantName,
		VisaType:      req.VisaType,
		Destination:   req.Destination,
		AssignedTo:    req.AssignedTo,
		Status:        Status(req.Status),
	}

	created, err := s.repo.Create(ctx, c)
	if err != nil {
		return VisaCase{}, mapRepoError(err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateVisaCaseRequest) (VisaCase, error) {
	if err := req.Validate(); err != nil {
		return VisaCase{}, err
	}
	if req.IsEmpty() {
		return VisaCase{}, response.NewAPIError(400, errNoFields)
	}

	var patch Patch
	if req.ApplicantName != nil {
		v := strings.TrimSpace(*req.ApplicantName)
		patch.ApplicantName = &v
	}
	if req.VisaType != nil {
		v := strings.TrimSpace(*req.VisaType)
		patch.VisaType = &v
	}
	if req.Destination != nil {
		v := strings.TrimSpace(*req.Destination)
		patch.Destination = &v
	}
	if req.AssignedTo != nil {
		v := strings.TrimSpace(*req.AssignedTo)
		patch.AssignedTo = &v
	}
	if req.Status != nil {
		status := Status(*req.Status)
		patch.Status = &status
	}

	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return VisaCase{}, mapRepoError(err)
	}
	return updated, nil
}

func mapRepoError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return response.NewAPIError(404, "visa case not found")
	}
	return err
}
