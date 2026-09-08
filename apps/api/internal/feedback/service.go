package feedback

import (
	"context"

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

func (s *Service) List(ctx context.Context, params pagination.Params) ([]Feedback, int, error) {
	return s.repo.List(ctx, params)
}

func (s *Service) Get(ctx context.Context, id int64) (Feedback, error) {
	fb, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Feedback{}, mapRepoError(err)
	}
	return fb, nil
}

// Create requires a booking. The caller may pass a customerID when known.
func (s *Service) Create(ctx context.Context, req CreateFeedbackRequest, customerID *int64) (Feedback, error) {
	if err := req.Validate(); err != nil {
		return Feedback{}, err
	}
	fb, err := s.repo.Create(ctx, Feedback{
		BookingID:  req.BookingID,
		CustomerID: customerID,
		Rating:     req.Rating,
		Comment:    req.Comment,
	})
	if err != nil {
		return Feedback{}, mapRepoError(err)
	}
	return fb, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateFeedbackRequest) (Feedback, error) {
	if err := req.Validate(); err != nil {
		return Feedback{}, err
	}
	if req.IsEmpty() {
		return Feedback{}, response.NewAPIError(400, "at least one field must be provided")
	}
	fb, err := s.repo.Update(ctx, id, req.Rating, req.Comment)
	if err != nil {
		return Feedback{}, mapRepoError(err)
	}
	return fb, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return mapRepoError(s.repo.Delete(ctx, id))
}

func mapRepoError(err error) error {
	return dberror.Map(err, ErrNotFound, "feedback not found")
}