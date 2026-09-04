package bookings

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]Booking, error) {
	return s.repo.List(ctx)
}

func (s *Service) Get(ctx context.Context, id int64) (Booking, error) {
	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Booking{}, mapRepoError(err)
	}
	return b, nil
}

func (s *Service) Create(ctx context.Context, req CreateBookingRequest) (Booking, error) {
	if err := req.Validate(); err != nil {
		return Booking{}, err
	}

	scheduledFor, err := time.Parse(time.RFC3339, req.ScheduledFor)
	if err != nil {
		return Booking{}, response.NewAPIError(400, "scheduledFor must be a valid RFC3339 timestamp")
	}

	b := Booking{
		CustomerName:    req.CustomerName,
		ServiceType:     ServiceType(req.ServiceType),
		ScheduledFor:    scheduledFor,
		DurationMinutes: req.DurationMinutes,
		Address:         req.Address,
		AssignedCleaner: req.AssignedCleaner,
		Status:          Status(req.Status),
		Notes:           req.Notes,
	}

	created, err := s.repo.Create(ctx, b)
	if err != nil {
		return Booking{}, mapRepoError(err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateBookingRequest) (Booking, error) {
	if err := req.Validate(); err != nil {
		return Booking{}, err
	}
	if req.IsEmpty() {
		return Booking{}, response.NewAPIError(400, errNoFields)
	}

	var patch Patch
	if req.CustomerName != nil {
		v := strings.TrimSpace(*req.CustomerName)
		patch.CustomerName = &v
	}
	if req.ServiceType != nil {
		st := ServiceType(*req.ServiceType)
		patch.ServiceType = &st
	}
	if req.ScheduledFor != nil {
		t, err := time.Parse(time.RFC3339, *req.ScheduledFor)
		if err != nil {
			return Booking{}, response.NewAPIError(400, "scheduledFor must be a valid RFC3339 timestamp")
		}
		patch.ScheduledFor = &t
	}
	if req.DurationMinutes != nil {
		patch.DurationMinutes = req.DurationMinutes
	}
	if req.Address != nil {
		v := strings.TrimSpace(*req.Address)
		patch.Address = &v
	}
	if req.AssignedCleaner != nil {
		v := strings.TrimSpace(*req.AssignedCleaner)
		patch.AssignedCleaner = &v
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
		return Booking{}, mapRepoError(err)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	return mapRepoError(err)
}

func mapRepoError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return response.NewAPIError(404, "booking not found")
	}
	return err
}
