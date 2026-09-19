package sites

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

func (s *Service) List(ctx context.Context, f Filters, params pagination.Params) ([]Site, int, error) {
	items, total, err := s.repo.List(ctx, f, params)
	if err != nil {
		return nil, 0, mapRepoError(err)
	}
	return items, total, nil
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
	site, err := s.repo.Create(ctx, Site{
		CustomerID:   req.CustomerID,
		Name:         req.Name,
		Address:      req.Address,
		Area:         req.Area,
		PropertyType: PropertyType(req.PropertyType),
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		ContactEmail: req.ContactEmail,
		Notes:        req.Notes,
		Status:       Status(req.Status),
		Lat:          req.Lat,
		Lng:          req.Lng,
	})
	if err != nil {
		return Site{}, mapRepoError(err)
	}
	return site, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateSiteRequest) (Site, error) {
	if err := req.Validate(); err != nil {
		return Site{}, err
	}
	if req.IsEmpty() {
		return Site{}, response.NewAPIError(400, "at least one field must be provided")
	}
	site, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return Site{}, mapRepoError(err)
	}
	return site, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return mapRepoError(s.repo.Delete(ctx, id))
}

// EnsureOwnedBy reports whether a site belongs to the given customer. It is the
// guard that stops one customer's booking being pointed at another's site.
func (s *Service) EnsureOwnedBy(ctx context.Context, siteID, customerID int64) error {
	owner, err := s.repo.OwnerOf(ctx, siteID)
	if err != nil {
		return mapRepoError(err)
	}
	if owner != customerID {
		return response.NewAPIError(422, "site belongs to a different customer")
	}
	return nil
}

func mapRepoError(err error) error {
	return dberror.Map(err, ErrNotFound, "site not found")
}
