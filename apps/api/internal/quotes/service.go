package quotes

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

func (s *Service) List(ctx context.Context, params pagination.Params, customerID int64) ([]Quote, int, error) {
	return s.repo.List(ctx, params, customerID)
}

func (s *Service) Get(ctx context.Context, id int64) (Quote, error) {
	q, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Quote{}, mapRepoError(err)
	}
	return q, nil
}

func (s *Service) Create(ctx context.Context, req CreateQuoteRequest) (Quote, error) {
	if err := req.Validate(); err != nil {
		return Quote{}, err
	}
	taxRate := 0.0
	if req.TaxRate != nil {
		taxRate = *req.TaxRate
	}
	sub, total := totals(req.Items, taxRate)
	q := Quote{
		CustomerID: req.CustomerID,
		SiteID:     req.SiteID,
		Status:     Status(req.Status),
		Subtotal:   sub,
		TaxRate:    taxRate,
		Total:      total,
		Currency:   strings.ToUpper(strings.TrimSpace(req.Currency)),
		Notes:      req.Notes,
	}
	if req.ValidUntil != nil {
		v, _ := parseDate(strings.TrimSpace(*req.ValidUntil))
		q.ValidUntil = &v
	}
	created, err := s.repo.Create(ctx, q, req.Items)
	if err != nil {
		return Quote{}, mapRepoError(err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateQuoteRequest, approve bool) (Quote, error) {
	if err := req.Validate(); err != nil {
		return Quote{}, err
	}
	if req.IsEmpty() && !approve {
		return Quote{}, response.NewAPIError(400, "at least one field must be provided")
	}
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Quote{}, mapRepoError(err)
	}
	var patch Patch
	patch.SiteID = req.SiteID
	if req.ClearSiteID != nil {
		patch.ClearSiteID = *req.ClearSiteID
	}
	if req.Status != nil {
		v := Status(strings.TrimSpace(*req.Status))
		// Terminal states are sticky: an accepted quote cannot go back to draft.
		if current.Status == StatusAccepted && v != StatusAccepted {
			return Quote{}, response.NewAPIError(422, "accepted quotes cannot be reopened")
		}
		patch.Status = &v
	}
	if approve {
		v := StatusAccepted
		if current.Status == StatusAccepted {
			return current, nil
		}
		patch.Status = &v
	}
	if req.ValidUntil != nil {
		v, _ := parseDate(strings.TrimSpace(*req.ValidUntil))
		patch.ValidUntil = &v
	}
	if req.ClearValidUntil != nil {
		patch.ClearValidUntil = *req.ClearValidUntil
	}
	patch.TaxRate = req.TaxRate
	if req.Currency != nil {
		v := strings.ToUpper(strings.TrimSpace(*req.Currency))
		patch.Currency = &v
	}
	if req.Notes != nil {
		v := strings.TrimSpace(*req.Notes)
		patch.Notes = &v
	}
	if req.Items != nil {
		patch.Items = req.Items
		patch.HasItems = true
	}
	patch.ConvertedBookingID = req.ConvertedBookingID
	patch.ConvertedContractID = req.ConvertedContractID
	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return Quote{}, mapRepoError(err)
	}
	return updated, nil
}

func mapRepoError(err error) error {
	return dberror.Map(err, ErrNotFound, "quote not found")
}
