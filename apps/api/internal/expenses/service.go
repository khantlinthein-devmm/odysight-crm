package expenses

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

func (s *Service) List(ctx context.Context, f Filters, params pagination.Params) ([]Expense, int, error) {
	items, total, err := s.repo.List(ctx, f, params)
	if err != nil {
		return nil, 0, mapRepoError(err)
	}
	return items, total, nil
}

// Create records one expense attributed to the acting user.
func (s *Service) Create(ctx context.Context, req CreateExpenseRequest, userID int64) (Expense, error) {
	if err := req.Validate(); err != nil {
		return Expense{}, err
	}
	e, err := s.repo.Create(ctx, Expense{
		SpentOn:   req.SpentOn,
		Category:  req.Category,
		Amount:    req.Amount,
		Note:      req.Note,
		CreatedBy: userID,
	})
	if err != nil {
		return Expense{}, mapRepoError(err)
	}
	return e, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateExpenseRequest) (Expense, error) {
	if err := req.Validate(); err != nil {
		return Expense{}, err
	}
	if req.IsEmpty() {
		return Expense{}, response.NewAPIError(400, "at least one field must be provided")
	}
	e, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return Expense{}, mapRepoError(err)
	}
	return e, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return mapRepoError(s.repo.Delete(ctx, id))
}

func mapRepoError(err error) error {
	return dberror.Map(err, ErrNotFound, "expense not found")
}
