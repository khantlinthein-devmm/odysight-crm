package attendance

import (
	"context"
	"errors"
	"time"

	"github.com/odysight/crm/pkg/dberror"
	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

// dateFormat is the YYYY-MM-DD layout used for attendance work dates.
const dateFormat = "2006-01-02"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, cleanerID int64, from, to string, params pagination.Params) ([]Record, int, error) {
	items, total, err := s.repo.List(ctx, cleanerID, from, to, params)
	if err != nil {
		return nil, 0, mapRepoError(err)
	}
	return items, total, nil
}

// CheckIn stamps the cleaner's arrival on the current work date. The date is
// taken from the server clock, so the API container's TZ decides the rollover
// hour (docker-compose sets Asia/Bangkok).
func (s *Service) CheckIn(ctx context.Context, req CheckActionRequest) (Record, error) {
	if err := req.Validate(); err != nil {
		return Record{}, err
	}
	rec, err := s.repo.CheckIn(ctx, req.CleanerID, today())
	if err != nil {
		return Record{}, mapRepoError(err)
	}
	return rec, nil
}

// CheckOut stamps the cleaner's departure on the current work date.
func (s *Service) CheckOut(ctx context.Context, req CheckActionRequest) (Record, error) {
	if err := req.Validate(); err != nil {
		return Record{}, err
	}
	rec, err := s.repo.CheckOut(ctx, req.CleanerID, today())
	if err != nil {
		return Record{}, mapRepoError(err)
	}
	return rec, nil
}

func today() string { return time.Now().Format(dateFormat) }

func mapRepoError(err error) error {
	switch {
	case errors.Is(err, ErrAlreadyCheckedIn):
		return response.NewAPIError(409, "already checked in for today")
	case errors.Is(err, ErrAlreadyCheckedOut):
		return response.NewAPIError(409, "already checked out for today")
	}
	return dberror.Map(err, ErrNotFound, "attendance record not found")
}
