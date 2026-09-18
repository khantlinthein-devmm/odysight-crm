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
		Area:      req.Area,
		UserID:    req.UserID,
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
	if req.Area != nil {
		v := strings.TrimSpace(*req.Area)
		patch.Area = &v
	}
	if req.UserID != nil {
		patch.UserID = req.UserID
	}
	if req.IsOnline != nil {
		patch.IsOnline = req.IsOnline
	}

	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return Cleaner{}, mapRepoError(err)
	}
	return updated, nil
}

// Me resolves the cleaner profile for the logged-in user (role CLEANER).
func (s *Service) Me(ctx context.Context, userID int64) (Cleaner, error) {
	c, err := s.repo.FindForUser(ctx, userID)
	if err != nil {
		return Cleaner{}, mapRepoError(err)
	}
	return c, nil
}

// UpdateLocation records a GPS ping for the logged-in cleaner.
func (s *Service) UpdateLocation(ctx context.Context, userID int64, req LocationUpdateRequest) (Cleaner, error) {
	if err := req.Validate(); err != nil {
		return Cleaner{}, err
	}
	c, err := s.repo.FindForUser(ctx, userID)
	if err != nil {
		return Cleaner{}, mapRepoError(err)
	}
	updated, err := s.repo.UpdateLocation(ctx, c.ID, req.Lat, req.Lng, req.Area, req.IsOnline)
	if err != nil {
		return Cleaner{}, mapRepoError(err)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	return mapRepoError(err)
}

// ListPhones returns all labeled phone numbers for a cleaner.
func (s *Service) ListPhones(ctx context.Context, cleanerID int64) ([]PhoneNumber, error) {
	phones, err := s.repo.ListPhones(ctx, cleanerID)
	if err != nil {
		return nil, mapRepoError(err)
	}
	return phones, nil
}

// GetPhones is an alias for ListPhones.
func (s *Service) GetPhones(ctx context.Context, cleanerID int64) ([]PhoneNumber, error) {
	return s.ListPhones(ctx, cleanerID)
}

// AddPhone validates and adds a labeled phone number to a cleaner.
func (s *Service) AddPhone(ctx context.Context, cleanerID int64, req AddPhoneRequest) (PhoneNumber, error) {
	if err := req.Validate(); err != nil {
		return PhoneNumber{}, err
	}
	phone, err := s.repo.AddPhone(ctx, cleanerID, req.Label, req.Phone)
	if err != nil {
		return PhoneNumber{}, mapRepoError(err)
	}
	return phone, nil
}

// RemovePhone deletes a labeled phone number scoped to its cleaner.
func (s *Service) RemovePhone(ctx context.Context, cleanerID, phoneID int64) error {
	return mapRepoError(s.repo.RemovePhone(ctx, cleanerID, phoneID))
}

func mapRepoError(err error) error {
	return dberror.Map(err, ErrNotFound, "cleaner not found")
}
