package audit

import (
	"context"

	"github.com/odysight/crm/pkg/pagination"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, params pagination.Params) ([]Entry, int, error) {
	return s.repo.List(ctx, params)
}
