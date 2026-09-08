package servicerecords

import (
	"context"
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

func (s *Service) List(ctx context.Context, params pagination.Params) ([]ServiceRecord, int, error) {
	return s.repo.List(ctx, params)
}

func (s *Service) Get(ctx context.Context, id int64) (ServiceRecord, error) {
	rec, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ServiceRecord{}, mapRepoError(err)
	}
	return rec, nil
}

func (s *Service) Create(ctx context.Context, req CreateServiceRecordRequest) (ServiceRecord, error) {
	if err := req.Validate(); err != nil {
		return ServiceRecord{}, err
	}

	rec := ServiceRecord{
		BookingNumber: req.BookingNumber,
		CleanerName:   req.CleanerName,
		ServiceType:   req.ServiceType,
		Rating:        req.Rating,
		Status:        Status(req.Status),
		Notes:         req.Notes,
	}

	created, err := s.repo.Create(ctx, rec)
	if err != nil {
		return ServiceRecord{}, mapRepoError(err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateServiceRecordRequest) (ServiceRecord, error) {
	if err := req.Validate(); err != nil {
		return ServiceRecord{}, err
	}
	if req.IsEmpty() {
		return ServiceRecord{}, response.NewAPIError(400, errNoFields)
	}

	var patch Patch
	if req.BookingNumber != nil {
		v := strings.TrimSpace(*req.BookingNumber)
		patch.BookingNumber = &v
	}
	if req.CleanerName != nil {
		v := strings.TrimSpace(*req.CleanerName)
		patch.CleanerName = &v
	}
	if req.ServiceType != nil {
		v := strings.TrimSpace(*req.ServiceType)
		patch.ServiceType = &v
	}
	if req.Rating != nil {
		patch.Rating = req.Rating
	}
	if req.Status != nil {
		status := Status(*req.Status)
		patch.Status = &status
	}
	if req.Notes != nil {
		v := strings.TrimSpace(*req.Notes)
		patch.Notes = &v
	}

	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return ServiceRecord{}, mapRepoError(err)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	return mapRepoError(err)
}

func mapRepoError(err error) error {
	return dberror.Map(err, ErrNotFound, "service record not found")
}
