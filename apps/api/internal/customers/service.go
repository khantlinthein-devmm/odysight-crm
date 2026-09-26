package customers

import (
	"context"
	"strings"

	"golang.org/x/crypto/bcrypt"

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

func (s *Service) List(ctx context.Context, params pagination.Params) ([]Customer, int, error) {
	return s.repo.List(ctx, params)
}

func (s *Service) Get(ctx context.Context, id int64) (Customer, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Customer{}, mapRepoError(err)
	}
	return c, nil
}

func (s *Service) Create(ctx context.Context, req CreateCustomerRequest) (Customer, error) {
	if err := req.Validate(); err != nil {
		return Customer{}, err
	}

	c := Customer{
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		Email:           req.Email,
		Phone:           req.Phone,
		Address:         req.Address,
		PropertyType:    PropertyType(req.PropertyType),
		Area:            req.Area,
		Status:          Status(req.Status),
		LeadID:          req.LeadID,
		TaxID:           req.TaxID,
		TaxBranch:       req.TaxBranch,
		WithholdingRate: req.WithholdingRate,
	}

	created, err := s.repo.Create(ctx, c)
	if err != nil {
		return Customer{}, mapRepoError(err)
	}
	return created, nil
}

// ConvertToCustomer creates a customer from data pre-filled by a converted lead.
// It skips the full create validation so callers can pass lead-derived values.
func (s *Service) ConvertToCustomer(ctx context.Context, req CreateCustomerRequest) (Customer, error) {
	if err := req.Validate(); err != nil {
		return Customer{}, err
	}

	c := Customer{
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		Email:           req.Email,
		Phone:           req.Phone,
		Address:         req.Address,
		PropertyType:    PropertyType(req.PropertyType),
		Area:            req.Area,
		Status:          Status(req.Status),
		LeadID:          req.LeadID,
		TaxID:           req.TaxID,
		TaxBranch:       req.TaxBranch,
		WithholdingRate: req.WithholdingRate,
	}

	created, err := s.repo.Create(ctx, c)
	if err != nil {
		return Customer{}, mapRepoError(err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateCustomerRequest) (Customer, error) {
	if err := req.Validate(); err != nil {
		return Customer{}, err
	}
	if req.IsEmpty() {
		return Customer{}, response.NewAPIError(400, errNoFields)
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
	if req.Address != nil {
		v := strings.TrimSpace(*req.Address)
		patch.Address = &v
	}
	if req.PropertyType != nil {
		pt := PropertyType(*req.PropertyType)
		patch.PropertyType = &pt
	}
	if req.Area != nil {
		v := strings.TrimSpace(*req.Area)
		patch.Area = &v
	}
	if req.Status != nil {
		status := Status(*req.Status)
		patch.Status = &status
	}
	patch.TaxID = req.TaxID
	patch.TaxBranch = req.TaxBranch
	patch.WithholdingRate = req.WithholdingRate

	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return Customer{}, mapRepoError(err)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	return mapRepoError(err)
}

// SetPortalAuth enables or disables a customer's portal access. A new
// password (when provided) replaces the stored hash before enabling.
func (s *Service) SetPortalAuth(ctx context.Context, id int64, password *string, enabled bool) (Customer, error) {
	if enabled && password != nil && len(*password) < 8 {
		return Customer{}, response.NewAPIError(400, "portal password must be at least 8 characters")
	}
	var hash *string
	if password != nil {
		h, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
		if err != nil {
			return Customer{}, response.NewAPIError(500, "failed to hash password")
		}
		hashed := string(h)
		hash = &hashed
	}
	// Disabling the portal clears the stored hash so it can never be used again.
	if !enabled {
		empty := ""
		hash = &empty
	}
	c, err := s.repo.UpdatePortalAuth(ctx, id, hash, enabled)
	if err != nil {
		return Customer{}, mapRepoError(err)
	}
	return c, nil
}

func mapRepoError(err error) error {
	return dberror.Map(err, ErrNotFound, "customer not found")
}
