package payments

import (
	"context"
	"errors"

	"github.com/odysight/crm/pkg/response"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]Payment, error) {
	return s.repo.List(ctx)
}

func (s *Service) Get(ctx context.Context, id int64) (Payment, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Payment{}, mapRepoError(err)
	}
	return p, nil
}

func (s *Service) Create(ctx context.Context, req CreatePaymentRequest) (Payment, error) {
	if err := req.Validate(); err != nil {
		return Payment{}, err
	}

	p := Payment{
		PayerName: req.PayerName,
		Amount:    req.Amount,
		Currency:  req.Currency,
		Method:    req.Method,
		Status:    Status(req.Status),
	}

	created, err := s.repo.Create(ctx, p)
	if err != nil {
		return Payment{}, mapRepoError(err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdatePaymentRequest) (Payment, error) {
	if err := req.Validate(); err != nil {
		return Payment{}, err
	}
	if req.IsEmpty() {
		return Payment{}, response.NewAPIError(400, errNoFields)
	}

	var patch Patch
	if req.Status != nil {
		status := Status(*req.Status)
		patch.Status = &status
	}
	patch.Method = req.Method

	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return Payment{}, mapRepoError(err)
	}
	return updated, nil
}

func mapRepoError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return response.NewAPIError(404, "payment not found")
	}
	return err
}
