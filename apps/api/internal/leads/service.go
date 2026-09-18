package leads

import (
	"context"
	"strings"

	"github.com/odysight/crm/internal/customers"
	"github.com/odysight/crm/pkg/dberror"
	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

type Service struct {
	repo      *Repository
	converter CustomerCreator
}

// CustomerCreator creates a customer record when a lead is converted.
type CustomerCreator interface {
	ConvertToCustomer(ctx context.Context, req customers.CreateCustomerRequest) (customers.Customer, error)
}

func NewService(repo *Repository, converter CustomerCreator) *Service {
	return &Service{repo: repo, converter: converter}
}

func (s *Service) List(ctx context.Context, params pagination.Params) ([]Lead, int, error) {
	return s.repo.List(ctx, params)
}

func (s *Service) Get(ctx context.Context, id int64) (Lead, error) {
	lead, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Lead{}, mapRepoError(err)
	}
	return lead, nil
}

func (s *Service) Create(ctx context.Context, req CreateLeadRequest) (Lead, error) {
	if err := req.Validate(); err != nil {
		return Lead{}, err
	}

	l := Lead{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Phone:     req.Phone,
		Status:    Status(req.Status),
		Source:    Source(req.Source),
	}

	created, err := s.repo.Create(ctx, l)
	if err != nil {
		return Lead{}, mapRepoError(err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateLeadRequest) (Lead, error) {
	if err := req.Validate(); err != nil {
		return Lead{}, err
	}
	if req.IsEmpty() {
		return Lead{}, response.NewAPIError(400, errNoFields)
	}

	var patch Patch
	if req.FirstName != nil {
		v := strings.TrimSpace(*req.FirstName)
		patch.FirstName = &v
	}
	if req.LastName != nil {
		v := strings.TrimSpace(*req.LastName)
		patch.LastName = &v
	}
	if req.Email != nil {
		patch.Email = req.Email
	}
	if req.Phone != nil {
		v := strings.TrimSpace(*req.Phone)
		patch.Phone = &v
	}
	if req.Status != nil {
		status := Status(*req.Status)
		patch.Status = &status
	}
	if req.Source != nil {
		source := Source(*req.Source)
		patch.Source = &source
	}

	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return Lead{}, mapRepoError(err)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	return mapRepoError(err)
}

// Convert turns a lead into a customer and marks the lead as won.
func (s *Service) Convert(ctx context.Context, id int64, req ConvertLeadRequest) (customers.Customer, error) {
	if err := req.Validate(); err != nil {
		return customers.Customer{}, err
	}

	lead, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return customers.Customer{}, mapRepoError(err)
	}
	if lead.Status == StatusLost || lead.Status == StatusWon {
		return customers.Customer{}, response.NewAPIError(400, "only open leads can be converted to a customer")
	}

	status := customers.Status(req.Status)
	createReq := customers.CreateCustomerRequest{
		FirstName:    lead.FirstName,
		LastName:     lead.LastName,
		Email:        lead.Email,
		Phone:        lead.Phone,
		Address:      req.Address,
		PropertyType: req.PropertyType,
		Area:         req.Area,
		Status:       string(status),
		LeadID:       &id,
	}

	created, err := s.converter.ConvertToCustomer(ctx, createReq)
	if err != nil {
		return customers.Customer{}, err
	}

	won := StatusWon
	_, err = s.repo.Update(ctx, id, Patch{Status: &won})
	if err != nil {
		return customers.Customer{}, mapRepoError(err)
	}

	return created, nil
}

func mapRepoError(err error) error {
	return dberror.Map(err, ErrNotFound, "lead not found")
}
