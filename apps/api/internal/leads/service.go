package leads

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

func (s *Service) List(ctx context.Context) ([]Lead, error) {
	return s.repo.List(ctx)
}

func (s *Service) Get(ctx context.Context, id int64) (Lead, error) {
	lead, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Lead{}, mapRepoError(err)
	}
	return lead, nil
}

func (s *Service) Create(ctx context.Context, req CreateLeadRequest) (Lead, error) {
	if err := req.Validate(); err != nil {
		return Lead{}, err
	}

	l := Lead{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Phone:     req.Phone,
		Status:    Status(req.Status),
		Source:    Source(req.Source),
	}

	created, err := s.repo.Create(ctx, l)
	if err != nil {
		return Lead{}, mapRepoError(err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateLeadRequest) (Lead, error) {
	if err := req.Validate(); err != nil {
		return Lead{}, err
	}
	if req.IsEmpty() {
		return Lead{}, response.NewAPIError(400, errNoFields)
	}

	var patch Patch
	if req.FirstName != nil {
		v := strings.TrimSpace(*req.FirstName)
		patch.FirstName = &v
	}
	if req.LastName != nil {
		v := strings.TrimSpace(*req.LastName)
		patch.LastName = &v
	}
	if req.Email != nil {
		patch.Email = req.Email
	}
	if req.Phone != nil {
		v := strings.TrimSpace(*req.Phone)
		patch.Phone = &v
	}
	if req.Status != nil {
		status := Status(*req.Status)
		patch.Status = &status
	}
	if req.Source != nil {
		source := Source(*req.Source)
		patch.Source = &source
	}

	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return Lead{}, mapRepoError(err)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	return mapRepoError(err)
}

func mapRepoError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return response.NewAPIError(404, "lead not found")
	}
	return err
}
