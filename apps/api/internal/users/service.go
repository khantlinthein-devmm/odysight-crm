package users

import (
	"context"
	"errors"
	"strings"

	"github.com/odysight/crm/internal/auth"
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

func (s *Service) List(ctx context.Context, params pagination.Params) ([]User, int, error) {
	return s.repo.List(ctx, params)
}

func (s *Service) Get(ctx context.Context, id int64) (User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return User{}, mapRepoError(err)
	}
	return u, nil
}

func (s *Service) Create(ctx context.Context, caller auth.Identity, req CreateUserRequest) (User, error) {
	if err := req.Validate(); err != nil {
		return User{}, err
	}
	// Only SUPER_ADMIN may create another SUPER_ADMIN.
	if auth.Role(req.Role) == auth.RoleSuperAdmin && caller.Role != auth.RoleSuperAdmin {
		return User{}, response.NewAPIError(403, "only SUPER_ADMIN can create SUPER_ADMIN users")
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return User{}, response.NewAPIError(500, "failed to hash password")
	}
	created, err := s.repo.Create(ctx, req.Name, req.Email, hash, req.Role)
	if err != nil {
		return User{}, mapCreateError(err, req.Email)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, caller auth.Identity, id int64, req UpdateUserRequest) (User, error) {
	if err := req.Validate(); err != nil {
		return User{}, err
	}
	if req.IsEmpty() {
		return User{}, response.NewAPIError(400, "no fields to update")
	}
	if req.Role != nil {
		if auth.Role(*req.Role) == auth.RoleSuperAdmin && caller.Role != auth.RoleSuperAdmin {
			return User{}, response.NewAPIError(403, "only SUPER_ADMIN can grant SUPER_ADMIN role")
		}
		if id == caller.UserID {
			return User{}, response.NewAPIError(400, "you cannot change your own role")
		}
	}
	if req.Name != nil {
		v := strings.TrimSpace(*req.Name)
		req.Name = &v
	}
	// Only SUPER_ADMIN may modify (including demote) a SUPER_ADMIN account.
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return User{}, mapRepoError(err)
	}
	if current.Role == string(auth.RoleSuperAdmin) && caller.Role != auth.RoleSuperAdmin {
		return User{}, response.NewAPIError(403, "only SUPER_ADMIN can modify SUPER_ADMIN users")
	}
	updated, err := s.repo.Update(ctx, id, Patch{Name: req.Name, Role: req.Role})
	if err != nil {
		return User{}, mapRepoError(err)
	}
	return updated, nil
}

func (s *Service) ResetPassword(ctx context.Context, caller auth.Identity, id int64, newPassword string) error {
	if len(newPassword) < 8 {
		return response.NewAPIError(400, "newPassword must be at least 8 characters")
	}
	target, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return mapRepoError(err)
	}
	// Only SUPER_ADMIN may reset another SUPER_ADMIN's password.
	if target.Role == string(auth.RoleSuperAdmin) && caller.Role != auth.RoleSuperAdmin {
		return response.NewAPIError(403, "only SUPER_ADMIN can reset SUPER_ADMIN passwords")
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return response.NewAPIError(500, "failed to hash password")
	}
	if err := s.repo.UpdatePassword(ctx, id, hash); err != nil {
		return mapRepoError(err)
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, caller auth.Identity, id int64) error {
	if id == caller.UserID {
		return response.NewAPIError(400, "you cannot delete your own account")
	}
	target, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return mapRepoError(err)
	}
	if target.Role == string(auth.RoleSuperAdmin) {
		if caller.Role != auth.RoleSuperAdmin {
			return response.NewAPIError(403, "only SUPER_ADMIN can delete SUPER_ADMIN users")
		}
		n, err := s.repo.CountByRole(ctx, string(auth.RoleSuperAdmin))
		if err != nil {
			return err
		}
		if n <= 1 {
			return response.NewAPIError(400, "you cannot delete the last SUPER_ADMIN")
		}
	}
	return mapRepoError(s.repo.Delete(ctx, id))
}

func mapRepoError(err error) error {
	return dberror.Map(err, ErrNotFound, "user not found")
}

func mapCreateError(err error, email string) error {
	mapped := dberror.Map(err, ErrNotFound, "user not found")
	var apiErr *response.APIError
	if errors.As(mapped, &apiErr) && apiErr.Status == 409 {
		return response.NewAPIError(409, "a user with email "+email+" already exists")
	}
	return mapped
}
