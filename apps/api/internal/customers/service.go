package customers

import (
	"context"
	"errors"
	"strings"

	"github.com/odysight/crm/pkg/response"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]Customer, error) {
	return s.repo.List(ctx)
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
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        req.Email,
		Phone:        req.Phone,
		Address:      req.Address,
		PropertyType: PropertyType(req.PropertyType),
		Area:         req.Area,
		Status:       Status(req.Status),
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

func mapRepoError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return response.NewAPIError(404, "customer not found")
	}
	return err
}
