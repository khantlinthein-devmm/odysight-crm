package applicants

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

func (s *Service) List(ctx context.Context) ([]Applicant, error) {
	return s.repo.List(ctx)
}

func (s *Service) Get(ctx context.Context, id int64) (Applicant, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Applicant{}, mapRepoError(err)
	}
	return a, nil
}

func (s *Service) Create(ctx context.Context, req CreateApplicantRequest) (Applicant, error) {
	if err := req.Validate(); err != nil {
		return Applicant{}, err
	}

	a := Applicant{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		Phone:       req.Phone,
		Nationality: req.Nationality,
		VisaType:    req.VisaType,
		Status:      Status(req.Status),
	}

	created, err := s.repo.Create(ctx, a)
	if err != nil {
		return Applicant{}, mapRepoError(err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateApplicantRequest) (Applicant, error) {
	if err := req.Validate(); err != nil {
		return Applicant{}, err
	}
	if req.IsEmpty() {
		return Applicant{}, response.NewAPIError(400, errNoFields)
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
	if req.Nationality != nil {
		v := strings.TrimSpace(*req.Nationality)
		patch.Nationality = &v
	}
	if req.VisaType != nil {
		v := strings.TrimSpace(*req.VisaType)
		patch.VisaType = &v
	}
	if req.Status != nil {
		status := Status(*req.Status)
		patch.Status = &status
	}

	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return Applicant{}, mapRepoError(err)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	return mapRepoError(err)
}

func mapRepoError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return response.NewAPIError(404, "applicant not found")
	}
	return err
}
