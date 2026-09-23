package sites

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

func (s *Service) List(ctx context.Context, params pagination.Params, customerID int64) ([]Site, int, error) {
	return s.repo.List(ctx, params, customerID)
}

func (s *Service) Get(ctx context.Context, id int64) (Site, error) {
	site, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Site{}, mapRepoError(err)
	}
	return site, nil
}

func (s *Service) Create(ctx context.Context, req CreateSiteRequest) (Site, error) {
	if err := req.Validate(); err != nil {
		return Site{}, err
	}
	site := Site{
		CustomerID:  req.CustomerID,
		Name:        req.Name,
		Address:     req.Address,
		ContactName: req.ContactName,
		Phone:       req.Phone,
		Email:       req.Email,
		Notes:       req.Notes,
		Status:      Status(req.Status),
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		IsDefault:   req.IsDefault,
	}
	created, err := s.repo.Create(ctx, site)
	if err != nil {
		return Site{}, mapRepoError(err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateSiteRequest) (Site, error) {
	if err := req.Validate(); err != nil {
		return Site{}, err
	}
	if req.IsEmpty() {
		return Site{}, response.NewAPIError(400, "at least one field must be provided")
	}
	var patch Patch
	if req.Name != nil {
		v := strings.TrimSpace(*req.Name)
		patch.Name = &v
	}
	if req.Address != nil {
		v := strings.TrimSpace(*req.Address)
		patch.Address = &v
	}
	if req.ContactName != nil {
		v := strings.TrimSpace(*req.ContactName)
		patch.ContactName = &v
	}
	if req.Phone != nil {
		v := strings.TrimSpace(*req.Phone)
		patch.Phone = &v
	}
	if req.Email != nil {
		patch.Email = req.Email
	}
	if req.Notes != nil {
		v := strings.TrimSpace(*req.Notes)
		patch.Notes = &v
	}
	if req.Status != nil {
		v := Status(strings.TrimSpace(*req.Status))
		patch.Status = &v
	}
	patch.Latitude = req.Latitude
	patch.Longitude = req.Longitude
	patch.IsDefault = req.IsDefault

	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return Site{}, mapRepoError(err)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return mapRepoError(s.repo.Delete(ctx, id))
}

func mapRepoError(err error) error {
	return dberror.Map(err, ErrNotFound, "site not found")
}
