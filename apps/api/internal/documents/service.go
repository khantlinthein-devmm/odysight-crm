package documents

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

func (s *Service) List(ctx context.Context) ([]Document, error) {
	return s.repo.List(ctx)
}

func (s *Service) Get(ctx context.Context, id int64) (Document, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Document{}, mapRepoError(err)
	}
	return d, nil
}

func (s *Service) Upload(ctx context.Context, req UploadDocumentRequest) (Document, error) {
	if err := req.Validate(); err != nil {
		return Document{}, err
	}

	d := Document{
		Name:          req.Name,
		Type:          req.Type,
		ApplicantName: req.ApplicantName,
		Status:        Status(req.Status),
	}

	created, err := s.repo.Create(ctx, d)
	if err != nil {
		return Document{}, mapRepoError(err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateDocumentRequest) (Document, error) {
	if err := req.Validate(); err != nil {
		return Document{}, err
	}
	if req.IsEmpty() {
		return Document{}, response.NewAPIError(400, errNoFields)
	}

	var patch Patch
	if req.Name != nil {
		v := strings.TrimSpace(*req.Name)
		patch.Name = &v
	}
	if req.Type != nil {
		v := strings.TrimSpace(*req.Type)
		patch.Type = &v
	}
	if req.Status != nil {
		status := Status(*req.Status)
		patch.Status = &status
	}

	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return Document{}, mapRepoError(err)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	return mapRepoError(err)
}

func mapRepoError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return response.NewAPIError(404, "document not found")
	}
	return err
}
