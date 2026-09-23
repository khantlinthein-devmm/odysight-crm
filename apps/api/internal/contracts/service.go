package contracts

import (
	"context"
	"strings"
	"time"

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

func (s *Service) List(ctx context.Context, params pagination.Params, customerID int64) ([]Contract, int, error) {
	return s.repo.List(ctx, params, customerID)
}

func (s *Service) Get(ctx context.Context, id int64) (Contract, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Contract{}, mapRepoError(err)
	}
	return c, nil
}

func (s *Service) Create(ctx context.Context, req CreateContractRequest) (Contract, error) {
	if err := req.Validate(); err != nil {
		return Contract{}, err
	}
	start, _ := parseDate(req.StartDate)
	end, _ := parseDate(req.EndDate)
	var renewal *time.Time
	if req.RenewalDate != nil {
		v, _ := parseDate(strings.TrimSpace(*req.RenewalDate))
		renewal = &v
	}
	c := Contract{
		CustomerID:       req.CustomerID,
		Title:            req.Title,
		Status:           Status(req.Status),
		StartDate:        start,
		EndDate:          end,
		RenewalDate:      renewal,
		ContractValue:    req.ContractValue,
		BillingFrequency: BillingFrequency(req.BillingFrequency),
		SLATerms:         req.SLATerms,
		Notes:            req.Notes,
		SiteIDs:          req.SiteIDs,
	}
	created, err := s.repo.Create(ctx, c)
	if err != nil {
		return Contract{}, mapRepoError(err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateContractRequest) (Contract, error) {
	if err := req.Validate(); err != nil {
		return Contract{}, err
	}
	if req.IsEmpty() {
		return Contract{}, response.NewAPIError(400, "at least one field must be provided")
	}
	if req.Status != nil {
		current, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return Contract{}, mapRepoError(err)
		}
		next := Status(strings.TrimSpace(*req.Status))
		if !canTransition(current.Status, next) {
			return Contract{}, response.NewAPIError(422, "cannot move contract from "+string(current.Status)+" to "+string(next))
		}
	}
	var patch Patch
	if req.Title != nil {
		v := strings.TrimSpace(*req.Title)
		patch.Title = &v
	}
	if req.Status != nil {
		v := Status(strings.TrimSpace(*req.Status))
		patch.Status = &v
	}
	if req.StartDate != nil {
		v, _ := parseDate(strings.TrimSpace(*req.StartDate))
		patch.StartDate = &v
	}
	if req.EndDate != nil {
		v, _ := parseDate(strings.TrimSpace(*req.EndDate))
		patch.EndDate = &v
	}
	if req.RenewalDate != nil {
		raw := strings.TrimSpace(*req.RenewalDate)
		if raw == "" {
			patch.ClearRenewal = true
		} else {
			v, _ := parseDate(raw)
			patch.RenewalDate = &v
		}
	}
	patch.ContractValue = req.ContractValue
	if req.BillingFrequency != nil {
		v := BillingFrequency(strings.TrimSpace(*req.BillingFrequency))
		patch.BillingFrequency = &v
	}
	if req.SLATerms != nil {
		v := strings.TrimSpace(*req.SLATerms)
		patch.SLATerms = &v
	}
	if req.Notes != nil {
		v := strings.TrimSpace(*req.Notes)
		patch.Notes = &v
	}
	if req.SiteIDs != nil || (req.ClearSiteIDs != nil && *req.ClearSiteIDs) {
		patch.ReplaceSites = true
		patch.SiteIDs = req.SiteIDs
		if patch.SiteIDs == nil {
			patch.SiteIDs = []int64{}
		}
	}
	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return Contract{}, mapRepoError(err)
	}
	return updated, nil
}

// allowedTransitions gates status edits so a contract's lifecycle stays
// meaningful: draft → active → expired, with cancelled/renewed terminal.
// Renewal never rewrites history — POST /contracts/:id/renew marks the old
// row renewed and opens a fresh draft covering the next period.
var allowedTransitions = map[Status][]Status{
	StatusDraft:     {StatusActive, StatusCancelled},
	StatusActive:    {StatusExpiring, StatusExpired, StatusCancelled},
	StatusExpiring:  {StatusActive, StatusExpired, StatusRenewed, StatusCancelled},
	StatusExpired:   {StatusRenewed},
	StatusCancelled: {},
	StatusRenewed:   {},
}

func canTransition(from, to Status) bool {
	if from == to {
		return true
	}
	for _, s := range allowedTransitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return mapRepoError(s.repo.Delete(ctx, id))
}

// RenewRequest opens the next period for a contract.
type RenewRequest struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

func (r *RenewRequest) Validate() error {
	r.StartDate = strings.TrimSpace(r.StartDate)
	r.EndDate = strings.TrimSpace(r.EndDate)
	start, err := parseDate(r.StartDate)
	if err != nil {
		return response.NewAPIError(400, "startDate must be YYYY-MM-DD")
	}
	end, err := parseDate(r.EndDate)
	if err != nil {
		return response.NewAPIError(400, "endDate must be YYYY-MM-DD")
	}
	if end.Before(start) {
		return response.NewAPIError(400, "endDate must be on or after startDate")
	}
	return nil
}

// Renew marks the contract renewed and opens a successor draft with the same
// customer, sites, value and billing frequency, atomically.
func (s *Service) Renew(ctx context.Context, id int64, req RenewRequest) (Contract, error) {
	if err := req.Validate(); err != nil {
		return Contract{}, err
	}
	start, _ := parseDate(req.StartDate)
	end, _ := parseDate(req.EndDate)
	next, err := s.repo.Renew(ctx, id, start, end)
	if err != nil {
		return Contract{}, mapRepoError(err)
	}
	return next, nil
}

func mapRepoError(err error) error {
	return dberror.Map(err, ErrNotFound, "contract not found")
}
