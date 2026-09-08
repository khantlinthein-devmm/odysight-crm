package cleaners

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

func (s *Service) List(ctx context.Context, params pagination.Params) ([]Cleaner, int, error) {
	return s.repo.List(ctx, params)
}

func (s *Service) Get(ctx context.Context, id int64) (Cleaner, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Cleaner{}, mapRepoError(err)
	}
	return c, nil
}

func (s *Service) Create(ctx context.Context, req CreateCleanerRequest) (Cleaner, error) {
	if err := req.Validate(); err != nil {
		return Cleaner{}, err
	}

	c := Cleaner{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		Email:     req.Email,
		Skills:    req.Skills,
		Status:    Status(req.Status),
	}

	created, err := s.repo.Create(ctx, c)
	if err != nil {
		return Cleaner{}, mapRepoError(err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateCleanerRequest) (Cleaner, error) {
	if err := req.Validate(); err != nil {
		return Cleaner{}, err
	}
	if req.IsEmpty() {
		return Cleaner{}, response.NewAPIError(400, errNoFields)
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
	if req.Phone != nil {
		v := strings.TrimSpace(*req.Phone)
		patch.Phone = &v
	}
	if req.Email != nil {
		patch.Email = req.Email
	}
	if req.Skills != nil {
		v := strings.TrimSpace(*req.Skills)
		patch.Skills = &v
	}
	if req.Status != nil {
		status := Status(*req.Status)
		patch.Status = &status
	}

	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return Cleaner{}, mapRepoError(err)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	return mapRepoError(err)
}

func mapRepoError(err error) error {
	return dberror.Map(err, ErrNotFound, "cleaner not found")
}
