package payments

import (
	"context"

	"github.com/odysight/crm/pkg/dberror"
	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

type Service struct {
	repo   *Repository
	ledger InvoiceLedger
}

// InvoiceLedger keeps invoices and receipts in step with payments: a paid
// payment credits the booking's invoice and issues a receipt, a refund
// cancels the receipt and re-opens the invoice.
type InvoiceLedger interface {
	// PayBooking records a paid payment against the booking's open invoice.
	// handled=false means there is no open invoice to pay.
	PayBooking(ctx context.Context, bookingNumber string, amount float64, method, reference string) (paymentID int64, handled bool, err error)
	SettlePayment(ctx context.Context, paymentID int64) error
	RefundPayment(ctx context.Context, paymentID int64) error
}

func NewService(repo *Repository, ledger InvoiceLedger) *Service {
	return &Service{repo: repo, ledger: ledger}
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

	// Money received for a booking with an open invoice goes through the
	// invoice ledger, which stores the payment and its receipt together.
	if Status(req.Status) == StatusPaid && s.ledger != nil {
		id, handled, err := s.ledger.PayBooking(ctx, req.BookingNumber, req.Amount, req.Method, req.Reference)
		if err != nil {
			return Payment{}, err
		}
		if handled {
			return s.Get(ctx, id)
		}
	}

	p := Payment{
		CustomerName:  req.CustomerName,
		BookingNumber: req.BookingNumber,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Method:        Method(req.Method),
		Status:        Status(req.Status),
		Reference:     req.Reference,
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

	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Payment{}, mapRepoError(err)
	}

	var next *Status
	if req.Status != nil {
		status := Status(*req.Status)
		if status != current.Status {
			if !allowedTransition(current.Status, status) {
				return Payment{}, response.NewAPIError(422,
					"cannot change payment from "+string(current.Status)+" to "+string(status))
			}
			next = &status
		}
	}

	var patch Patch
	if req.Method != nil {
		method := Method(*req.Method)
		patch.Method = &method
	}
	// Paid and refunded go through the ledger so the invoice balance and the
	// receipts move in the same transaction as the payment status.
	viaLedger := next != nil && s.ledger != nil && (*next == StatusPaid || *next == StatusRefunded)
	if next != nil && !viaLedger {
		patch.Status = next
	}
	if patch.Status != nil || patch.Method != nil {
		if _, err := s.repo.Update(ctx, id, patch); err != nil {
			return Payment{}, mapRepoError(err)
		}
	}
	if viaLedger {
		if *next == StatusPaid {
			err = s.ledger.SettlePayment(ctx, id)
		} else {
			err = s.ledger.RefundPayment(ctx, id)
		}
		if err != nil {
			return Payment{}, err
		}
	}
	return s.Get(ctx, id)
}

// allowedTransition guards the money trail: paid money can only move to
// refunded (never back to pending/failed), and refunded is terminal.
func allowedTransition(from, to Status) bool {
	switch from {
	case StatusPending:
		return to == StatusPaid || to == StatusFailed
	case StatusFailed:
		return to == StatusPending || to == StatusPaid
	case StatusPaid:
		return to == StatusRefunded
	default:
		return false
	}
}

func mapRepoError(err error) error {
	return dberror.Map(err, ErrNotFound, "payment not found")
}
