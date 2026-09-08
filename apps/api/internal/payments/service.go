package payments

import (
	"context"

	"github.com/odysight/crm/pkg/dberror"
	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

type Service struct {
	repo    *Repository
	settles InvoiceSettler
}

// InvoiceSettler marks invoices paid when a payment reaches the paid status.
type InvoiceSettler interface {
	MarkPaidForBooking(ctx context.Context, bookingNumber string) error
}

func NewService(repo *Repository, settles InvoiceSettler) *Service {
	return &Service{repo: repo, settles: settles}
}

func (s *Service) List(ctx context.Context, params pagination.Params) ([]Payment, int, error) {
	return s.repo.List(ctx, params)
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
		CustomerName:  req.CustomerName,
		BookingNumber: req.BookingNumber,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Method:        Method(req.Method),
		Status:        Status(req.Status),
	}

	created, err := s.repo.Create(ctx, p)
	if err != nil {
		return Payment{}, mapRepoError(err)
	}
	if created.Status == StatusPaid && s.settles != nil {
		if settleErr := s.settles.MarkPaidForBooking(ctx, created.BookingNumber); settleErr != nil {
			return Payment{}, settleErr
		}
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
	if req.Method != nil {
		method := Method(*req.Method)
		patch.Method = &method
	}

	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return Payment{}, mapRepoError(err)
	}
	if updated.Status == StatusPaid && s.settles != nil {
		if settleErr := s.settles.MarkPaidForBooking(ctx, updated.BookingNumber); settleErr != nil {
			return Payment{}, settleErr
		}
	}
	return updated, nil
}

func mapRepoError(err error) error {
	return dberror.Map(err, ErrNotFound, "payment not found")
}
